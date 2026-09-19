import type { CertificateVerificationTranslations } from "./locales/zh"

type VerificationCopy = CertificateVerificationTranslations
type VerificationErrors = VerificationCopy["errors"]

function formatCopy(template: string, values: Record<string, string | number>) {
  return template.replace(/\{\{(\w+)\}\}/g, (_, key: string) => String(values[key] ?? ""))
}

const ULID_PATTERN = /^[0-7][0-9A-HJKMNP-TV-Z]{25}$/i
const VERIFICATION_API_BASE = "/api/public/test-verify-creds"

export type VerificationStepId = "parse" | "digest" | "chain" | "certificate" | "signature" | "identity" | "lifecycle"
export type VerificationStepStatus = "pending" | "running" | "success" | "failed"

export type VerificationStep = {
  id: VerificationStepId
  status: VerificationStepStatus
  detail: string
}

export type VerificationProgress = (step: VerificationStep) => void

export type RootTrustAnchor = {
  public_key_pem: string
  algorithm: string
  key_id?: string
  revoked_leaf_fingerprints?: string[]
}

export type LifecycleResponse = {
  status?: number | string
  exists?: boolean
  is_valid?: boolean
  cred_def_name?: string
  issue_date?: string
  valid_until?: string
  revoked_at?: string
  revoke_reason?: string
  [key: string]: unknown
}

export type CredentialVerificationResult = {
  lifecycle: LifecycleResponse
  ulid: string
  pdfHash: string
  signerSubject: string
  leafFingerprint: string
  cryptoSuite: string
  rootKeyId: string
  leafValidFrom?: Date
  leafValidTo?: Date
}

type ASN1Element = {
  tag: number
  length: number
  headerLen: number
  rawBytes: Uint8Array
  contentBytes: Uint8Array
  children: ASN1Element[]
}

let rootAnchorPromise: Promise<RootTrustAnchor> | null = null
let rootCryptoKeyPromise: Promise<CryptoKey> | null = null

function verificationUrl(path: string) {
  return `${VERIFICATION_API_BASE}${path}`
}

function hex(bytes: ArrayBuffer | Uint8Array) {
  return Array.from(new Uint8Array(bytes instanceof Uint8Array ? bytes : bytes)).map((value) => value.toString(16).padStart(2, "0")).join("")
}

function decodeLatin1(bytes: Uint8Array) {
  return new TextDecoder("latin1").decode(bytes)
}

function asArrayBuffer(bytes: Uint8Array): ArrayBuffer {
  return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength) as ArrayBuffer
}

function parseASN1(bytes: Uint8Array, errors: VerificationErrors, offset = 0, limit = bytes.length): ASN1Element {
  if (offset + 2 > limit) throw new Error(errors.asn1Incomplete)
  const tag = bytes[offset]
  const firstLengthByte = bytes[offset + 1]
  let length = 0
  let headerLen = 2

  if (firstLengthByte < 0x80) {
    length = firstLengthByte
  } else {
    const lengthBytes = firstLengthByte & 0x7f
    if (lengthBytes === 0 || lengthBytes > 4 || offset + 2 + lengthBytes > limit) throw new Error(errors.asn1LengthInvalid)
    headerLen += lengthBytes
    for (let index = 0; index < lengthBytes; index += 1) length = (length << 8) | bytes[offset + 2 + index]
  }

  const contentStart = offset + headerLen
  const contentEnd = contentStart + length
  if (contentEnd > limit) throw new Error(errors.asn1OutOfBounds)
  const rawBytes = bytes.slice(offset, contentEnd)
  const contentBytes = bytes.slice(contentStart, contentEnd)
  const element: ASN1Element = { tag, length, headerLen, rawBytes, contentBytes, children: [] }

  if ((tag & 0x20) !== 0 || (tag >= 0xa0 && tag <= 0xbf)) {
    let childOffset = 0
    while (childOffset < contentBytes.length) {
      const child = parseASN1(contentBytes, errors, childOffset, contentBytes.length)
      element.children.push(child)
      childOffset += child.rawBytes.length
    }
  }
  return element
}

function encodeLength(length: number) {
  if (length < 0x80) return new Uint8Array([length])
  const values: number[] = []
  let remaining = length
  while (remaining > 0) {
    values.unshift(remaining & 0xff)
    remaining >>>= 8
  }
  return new Uint8Array([0x80 | values.length, ...values])
}

function replaceImplicitSetTag(element: ASN1Element) {
  const lengthBytes = encodeLength(element.contentBytes.length)
  const output = new Uint8Array(1 + lengthBytes.length + element.contentBytes.length)
  output[0] = 0x31
  output.set(lengthBytes, 1)
  output.set(element.contentBytes, 1 + lengthBytes.length)
  return output
}

function derSignatureToP1363(signature: Uint8Array, keySize: number, errors: VerificationErrors) {
  const parsed = parseASN1(signature, errors)
  if (parsed.tag !== 0x30 || parsed.children.length < 2) throw new Error(errors.ecdsaFormatInvalid)
  const output = new Uint8Array(keySize * 2)
  for (const [index, integer] of parsed.children.slice(0, 2).entries()) {
    let value = integer.contentBytes
    while (value.length > keySize && value[0] === 0) value = value.slice(1)
    if (value.length > keySize) throw new Error(errors.ecdsaValueTooLong)
    output.set(value, index * keySize + keySize - value.length)
  }
  return output
}

function decodeOID(bytes: Uint8Array, errors: VerificationErrors) {
  if (!bytes.length) return ""
  const first = bytes[0]
  const parts = [Math.min(2, Math.floor(first / 40)), first >= 80 ? first - 80 : first % 40]
  let value = 0
  for (const byte of bytes.slice(1)) {
    value = (value << 7) | (byte & 0x7f)
    if ((byte & 0x80) === 0) {
      parts.push(value)
      value = 0
    }
  }
  if (value !== 0) throw new Error(errors.oidIncomplete)
  return parts.join(".")
}

function hashNameForOID(oid: string, errors: VerificationErrors) {
  if (oid === "2.16.840.1.101.3.4.2.2" || oid === "1.2.840.10045.4.3.3") return "SHA-384" as const
  if (oid === "2.16.840.1.101.3.4.2.3" || oid === "1.2.840.10045.4.3.4") return "SHA-512" as const
  if (oid === "2.16.840.1.101.3.4.2.1" || oid === "1.2.840.10045.4.3.2") return "SHA-256" as const
  throw new Error(formatCopy(errors.unsupportedDigestOID, { oid }))
}

function curveForDigestOID(oid: string, errors: VerificationErrors) {
  // gcreds pairs the SignerInfo digest OID with the signing curve:
  // P-256 -> SHA-256, P-384 -> SHA-384, P-521 -> SHA-512.
  if (oid === "2.16.840.1.101.3.4.2.2") return { name: "P-384" as const, size: 48 }
  if (oid === "2.16.840.1.101.3.4.2.3") return { name: "P-521" as const, size: 66 }
  if (oid === "2.16.840.1.101.3.4.2.1") return { name: "P-256" as const, size: 32 }
  throw new Error(formatCopy(errors.unsupportedCurveDigestOID, { oid }))
}

function findSubjectPublicKeyInfo(tbs: ASN1Element) {
  return tbs.children.find((child) => child.tag === 0x30 && child.children.length === 2 && child.children[1].tag === 0x03)
}

function readASN1String(element: ASN1Element) {
  return decodeLatin1(element.contentBytes).replace(/\0/g, "").trim()
}

function subjectCommonName(tbs: ASN1Element, errors: VerificationErrors) {
  for (const child of tbs.children) {
    if (child.tag !== 0x30) continue
    for (const rdn of child.children) {
      const attribute = rdn.children[0]
      if (!attribute || attribute.children.length < 2) continue
      const oid = decodeOID(attribute.children[0].contentBytes, errors)
      if (oid === "2.5.4.3") return readASN1String(attribute.children[1])
    }
  }
  return ""
}

function parseCertificateTimes(tbs: ASN1Element) {
  const versionOffset = tbs.children[0]?.tag === 0xa0 ? 1 : 0
  const validity = tbs.children[versionOffset + 4]
  if (!validity || validity.tag !== 0x30 || validity.children.length < 2) return {}
  const parseTime = (element: ASN1Element) => {
    const raw = readASN1String(element)
    const normalized = element.tag === 0x17
      ? `${Number(raw.slice(0, 2)) >= 50 ? "19" : "20"}${raw}`
      : raw
    const iso = normalized.replace(/(\d{4})(\d{2})(\d{2})(\d{2})(\d{2})(\d{2})Z/, "$1-$2-$3T$4:$5:$6Z")
    const date = new Date(iso)
    return Number.isNaN(date.getTime()) ? undefined : date
  }
  return { from: parseTime(validity.children[0]), to: parseTime(validity.children[1]) }
}

function importPEMPublicKey(pem: string, algorithm: string, errors: VerificationErrors) {
  const clean = pem.replace(/-----[^\n]+-----/g, "").replace(/\s+/g, "")
  const der = Uint8Array.from(atob(clean), (char) => char.charCodeAt(0))
  const parsed = parseASN1(der, errors)
  let spki = der
  if (parsed.tag === 0x30 && parsed.children.length === 3) {
    spki = (findSubjectPublicKeyInfo(parsed.children[0])?.rawBytes || der) as Uint8Array<ArrayBuffer>
  }
  const curve = algorithm.includes("384") ? "P-384" : algorithm.includes("521") ? "P-521" : "P-256"
  return crypto.subtle.importKey("spki", asArrayBuffer(spki), { name: "ECDSA", namedCurve: curve }, false, ["verify"])
}

async function getRootAnchor(copy: VerificationCopy) {
  if (!rootAnchorPromise) {
    rootAnchorPromise = fetch(verificationUrl("/api/primary-key"), { credentials: "include", headers: { Accept: "application/json" } }).then(async (response) => {
      const payload = await response.json().catch(() => null) as { data?: RootTrustAnchor; message?: string } | null
      if (!response.ok || !payload?.data) throw new Error(payload?.message || formatCopy(copy.errors.rootAnchorFetch, { status: response.status }))
      const anchor = payload.data
      if (!anchor.public_key_pem || !anchor.algorithm) throw new Error(copy.errors.rootAnchorFieldsMissing)
      return anchor
    }).catch((error) => {
      rootAnchorPromise = null
      throw error
    })
  }
  return rootAnchorPromise
}

async function getRootKey(anchor: RootTrustAnchor, errors: VerificationErrors) {
  if (!rootCryptoKeyPromise) {
    rootCryptoKeyPromise = importPEMPublicKey(anchor.public_key_pem, anchor.algorithm, errors).catch((error) => {
      rootCryptoKeyPromise = null
      throw error
    })
  }
  return rootCryptoKeyPromise
}

function step(progress: VerificationProgress | undefined, id: VerificationStepId, status: VerificationStepStatus, detail: string) {
  progress?.({ id, status, detail })
}

function requireRange(bytes: Uint8Array, start: number, length: number, errors: VerificationErrors) {
  if (!Number.isSafeInteger(start) || !Number.isSafeInteger(length) || start < 0 || length < 0 || start + length > bytes.length) {
    throw new Error(errors.byteRangeOutOfBounds)
  }
}

export function isActiveLifecycleStatus(status: unknown) {
  return status === 1 || status === "Active" || status === "CREDENTIAL_STATUS_ACTIVE"
}

export function isRevokedLifecycleStatus(status: unknown) {
  return status === 2 || status === "Revoked" || status === "CREDENTIAL_STATUS_REVOKED"
}

export function isExpiredLifecycleStatus(status: unknown) {
  return status === 3 || status === "Expired" || status === "CREDENTIAL_STATUS_EXPIRED"
}

export function lifecycleStatusLabel(status: unknown, exists: boolean | undefined, labels: VerificationCopy["status"]) {
  if (exists === false) return labels.missing
  if (isActiveLifecycleStatus(status)) return labels.active
  if (isRevokedLifecycleStatus(status)) return labels.revoked
  if (isExpiredLifecycleStatus(status)) return labels.expired
  if (status === 0 || status === "Unspecified" || status === "CREDENTIAL_STATUS_UNSPECIFIED") return labels.unspecified
  return labels.unknown
}

export async function verifyCredentialPdf(file: File, progress: VerificationProgress | undefined, copy: VerificationCopy): Promise<CredentialVerificationResult> {
  const errors = copy.errors
  if (file.type && file.type !== "application/pdf") throw new Error(errors.pdfRequired)
  const bytes = new Uint8Array(await file.arrayBuffer())
  if (bytes.length < 32) throw new Error(errors.pdfTooShort)
  const anchor = await getRootAnchor(copy)

  step(progress, "parse", "running", copy.progress.parseRunning)
  const pdfText = decodeLatin1(bytes)
  if (!pdfText.startsWith("%PDF-")) throw new Error(errors.invalidPdf)
  const rangeMatch = /\/ByteRange\s*\[\s*(\d+)\s+(\d+)\s+(\d+)\s+(\d+)\s*\]/.exec(pdfText)
  if (!rangeMatch) throw new Error(errors.byteRangeMissing)
  const ranges = rangeMatch.slice(1).map(Number)
  requireRange(bytes, ranges[0], ranges[1], errors)
  requireRange(bytes, ranges[2], ranges[3], errors)
  const contentsMatch = /\/Contents\s*<([0-9A-Fa-f\s]+)>/.exec(pdfText)
  if (!contentsMatch) throw new Error(errors.contentsMissing)
  const signatureHex = contentsMatch[1].replace(/\s/g, "")
  if (!signatureHex || signatureHex.length % 2 !== 0) throw new Error(errors.contentsInvalid)
  const signatureContainer = Uint8Array.from((signatureHex.match(/.{2}/g) || []).map((value) => Number.parseInt(value, 16)))
  const p7Root = parseASN1(signatureContainer, errors)
  step(progress, "parse", "success", formatCopy(copy.progress.parseSuccess, { r1Start: ranges[0], r1Length: ranges[1], r2Start: ranges[2], r2Length: ranges[3] }))

  step(progress, "digest", "running", copy.progress.digestRunning)
  const contentBytes = new Uint8Array(ranges[1] + ranges[3])
  contentBytes.set(bytes.subarray(ranges[0], ranges[0] + ranges[1]), 0)
  contentBytes.set(bytes.subarray(ranges[2], ranges[2] + ranges[3]), ranges[1])

  const contentInfo = p7Root.children[1]
  const signedData = contentInfo?.children[0]
  if (!signedData) throw new Error(errors.signedDataInvalid)
  const certificates = signedData.children.find((child) => child.tag === 0xa0)
  // SignedData contains two SET fields: digestAlgorithms first and signerInfos
  // last. The golden verifier uses the CMS field order to select signerInfos.
  let signerInfos: ASN1Element | undefined
  for (const child of signedData.children) {
    if (child.tag === 0x31) signerInfos = child
  }
  const leafCertificate = certificates?.children.find((child) => child.tag === 0x30)
  const signerInfo = signerInfos?.children[0]
  if (!leafCertificate || !signerInfo) throw new Error(errors.signerMissing)
  const signerDigestAlgorithm = signerInfo.children.find((child) => child.tag === 0x30 && child.children[0]?.tag === 0x06)
  const authenticatedAttributes = signerInfo.children.find((child) => child.tag === 0xa0)
  const signature = signerInfo.children.find((child) => child.tag === 0x04)?.contentBytes
  if (!signerDigestAlgorithm || !authenticatedAttributes || !signature) throw new Error(errors.signerFieldsMissing)
  const digestOID = decodeOID(signerDigestAlgorithm.children[0].contentBytes, errors)
  const hashName = hashNameForOID(digestOID, errors)
  const computedDigest = hex(await crypto.subtle.digest(hashName, contentBytes))
  step(progress, "digest", "success", formatCopy(copy.progress.digestSuccess, { hashName, digest: computedDigest.slice(0, 16) }))

  step(progress, "chain", "running", copy.progress.chainRunning)
  const leafFingerprint = hex(await crypto.subtle.digest("SHA-256", asArrayBuffer(leafCertificate.rawBytes)))
  if ((anchor.revoked_leaf_fingerprints || []).some((item) => item.toLowerCase() === leafFingerprint.toLowerCase())) {
    throw new Error(formatCopy(errors.leafRevoked, { fingerprint: leafFingerprint }))
  }
  const leafTBS = leafCertificate.children[0]
  const leafSignatureAlgorithm = leafCertificate.children[1]
  const leafSignatureValue = leafCertificate.children[2]?.contentBytes.slice(1)
  const leafSPKI = findSubjectPublicKeyInfo(leafTBS)
  if (!leafTBS || !leafSignatureAlgorithm || !leafSignatureValue || !leafSPKI) throw new Error(errors.certificateInvalid)
  const curve = curveForDigestOID(digestOID, errors)
  const leafSignatureOID = decodeOID(leafSignatureAlgorithm.children[0].contentBytes, errors)
  const leafSignatureHash = hashNameForOID(leafSignatureOID, errors)
  const trusted = await crypto.subtle.verify(
    { name: "ECDSA", hash: leafSignatureHash },
    await getRootKey(anchor, errors),
    derSignatureToP1363(leafSignatureValue, curve.size, errors),
    asArrayBuffer(leafTBS.rawBytes),
  )
  if (!trusted) throw new Error(errors.certificateUntrusted)
  const validity = parseCertificateTimes(leafTBS)
  const now = Date.now()
  if (validity.from && now < validity.from.getTime()) throw new Error(errors.certificateNotYetValid)
  if (validity.to && now > validity.to.getTime()) throw new Error(errors.certificateExpired)
  step(progress, "chain", "success", formatCopy(copy.progress.chainSuccess, { fingerprint: leafFingerprint.slice(0, 16) }))

  step(progress, "certificate", "success", formatCopy(copy.progress.certificateSuccess, { subject: subjectCommonName(leafTBS, errors), curve: curve.name }))
  step(progress, "signature", "running", copy.progress.signatureRunning)
  const signedAttributes = replaceImplicitSetTag(authenticatedAttributes)
  const leafKey = await crypto.subtle.importKey("spki", asArrayBuffer(leafSPKI.rawBytes), { name: "ECDSA", namedCurve: curve.name }, false, ["verify"])
  const signatureValid = await crypto.subtle.verify(
    { name: "ECDSA", hash: hashName },
    leafKey,
    derSignatureToP1363(signature, curve.size, errors),
    asArrayBuffer(signedAttributes),
  )
  if (!signatureValid) throw new Error(errors.signatureInvalid)

  let embeddedDigest = ""
  let ulid = ""
  for (const attribute of authenticatedAttributes.children) {
    if (attribute.children.length < 2 || attribute.children[0].tag !== 0x06) continue
    const oid = decodeOID(attribute.children[0].contentBytes, errors)
    const value = attribute.children[1].children[0]
    if (!value) continue
    if (oid === "1.2.840.113549.1.9.4") embeddedDigest = hex(value.contentBytes)
    if (oid === "1.3.6.1.4.1.99999.1") ulid = readASN1String(value).toUpperCase()
  }
  if (embeddedDigest.toLowerCase() !== computedDigest.toLowerCase()) throw new Error(errors.digestMismatch)
  step(progress, "signature", "success", copy.progress.signatureSuccess)
  if (!ULID_PATTERN.test(ulid)) throw new Error(errors.ulidMissing)
  step(progress, "identity", "success", formatCopy(copy.progress.identitySuccess, { ulid }))

  step(progress, "lifecycle", "running", copy.progress.lifecycleRunning)
  const pdfHash = hex(await crypto.subtle.digest("SHA-256", asArrayBuffer(bytes)))
  const response = await fetch(verificationUrl(`/api/check-validity?cred_ulid=${encodeURIComponent(ulid)}&pdf_hash=${encodeURIComponent(pdfHash)}`), { credentials: "include", headers: { Accept: "application/json" } })
  const payload = await response.json().catch(() => null) as { data?: LifecycleResponse; message?: string } | null
  if (!response.ok || !payload?.data) throw new Error(payload?.message || formatCopy(errors.onlineCheckFailed, { status: response.status }))
  const lifecycle = payload.data
  step(progress, "lifecycle", "success", formatCopy(copy.progress.lifecycleSuccess, { status: lifecycleStatusLabel(lifecycle.status, lifecycle.exists, copy.status) }))
  return {
    lifecycle,
    ulid,
    pdfHash,
    signerSubject: subjectCommonName(leafTBS, errors),
    leafFingerprint,
    cryptoSuite: `${curve.name} / ${hashName}`,
    rootKeyId: anchor.key_id || "",
    leafValidFrom: validity.from,
    leafValidTo: validity.to,
  }
}
