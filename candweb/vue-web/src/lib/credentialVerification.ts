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

function parseASN1(bytes: Uint8Array, offset = 0, limit = bytes.length): ASN1Element {
  if (offset + 2 > limit) throw new Error("ASN.1 数据不完整")
  const tag = bytes[offset]
  const firstLengthByte = bytes[offset + 1]
  let length = 0
  let headerLen = 2

  if (firstLengthByte < 0x80) {
    length = firstLengthByte
  } else {
    const lengthBytes = firstLengthByte & 0x7f
    if (lengthBytes === 0 || lengthBytes > 4 || offset + 2 + lengthBytes > limit) throw new Error("ASN.1 长度字段无效")
    headerLen += lengthBytes
    for (let index = 0; index < lengthBytes; index += 1) length = (length << 8) | bytes[offset + 2 + index]
  }

  const contentStart = offset + headerLen
  const contentEnd = contentStart + length
  if (contentEnd > limit) throw new Error("ASN.1 内容超出边界")
  const rawBytes = bytes.slice(offset, contentEnd)
  const contentBytes = bytes.slice(contentStart, contentEnd)
  const element: ASN1Element = { tag, length, headerLen, rawBytes, contentBytes, children: [] }

  if ((tag & 0x20) !== 0 || (tag >= 0xa0 && tag <= 0xbf)) {
    let childOffset = 0
    while (childOffset < contentBytes.length) {
      const child = parseASN1(contentBytes, childOffset, contentBytes.length)
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

function derSignatureToP1363(signature: Uint8Array, keySize: number) {
  const parsed = parseASN1(signature)
  if (parsed.tag !== 0x30 || parsed.children.length < 2) throw new Error("ECDSA 签名格式无效")
  const output = new Uint8Array(keySize * 2)
  for (const [index, integer] of parsed.children.slice(0, 2).entries()) {
    let value = integer.contentBytes
    while (value.length > keySize && value[0] === 0) value = value.slice(1)
    if (value.length > keySize) throw new Error("ECDSA 签名数值超出曲线长度")
    output.set(value, index * keySize + keySize - value.length)
  }
  return output
}

function decodeOID(bytes: Uint8Array) {
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
  if (value !== 0) throw new Error("OID 数据不完整")
  return parts.join(".")
}

function hashNameForOID(oid: string) {
  if (oid === "2.16.840.1.101.3.4.2.2" || oid === "1.2.840.10045.4.3.3") return "SHA-384" as const
  if (oid === "2.16.840.1.101.3.4.2.3" || oid === "1.2.840.10045.4.3.4") return "SHA-512" as const
  if (oid === "2.16.840.1.101.3.4.2.1" || oid === "1.2.840.10045.4.3.2") return "SHA-256" as const
  throw new Error(`不支持的摘要算法 OID: ${oid}`)
}

function curveForOID(oid: string) {
  if (oid === "1.3.132.0.34" || oid === "1.2.840.10045.3.1.8") return { name: "P-384" as const, size: 48 }
  if (oid === "1.3.132.0.35" || oid === "1.2.840.10045.3.1.9") return { name: "P-521" as const, size: 66 }
  if (oid === "1.2.840.10045.3.1.7") return { name: "P-256" as const, size: 32 }
  throw new Error(`不支持的椭圆曲线 OID: ${oid}`)
}

function findSubjectPublicKeyInfo(tbs: ASN1Element) {
  return tbs.children.find((child) => child.tag === 0x30 && child.children.length === 2 && child.children[1].tag === 0x03)
}

function readASN1String(element: ASN1Element) {
  return decodeLatin1(element.contentBytes).replace(/\0/g, "").trim()
}

function subjectCommonName(tbs: ASN1Element) {
  for (const child of tbs.children) {
    if (child.tag !== 0x30) continue
    for (const rdn of child.children) {
      const attribute = rdn.children[0]
      if (!attribute || attribute.children.length < 2) continue
      const oid = decodeOID(attribute.children[0].contentBytes)
      if (oid === "2.5.4.3") return readASN1String(attribute.children[1])
    }
  }
  return "官方签名证书"
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

function importPEMPublicKey(pem: string, algorithm: string) {
  const clean = pem.replace(/-----[^\n]+-----/g, "").replace(/\s+/g, "")
  const der = Uint8Array.from(atob(clean), (char) => char.charCodeAt(0))
  const parsed = parseASN1(der)
  let spki = der
  if (parsed.tag === 0x30 && parsed.children.length === 3) {
    spki = (findSubjectPublicKeyInfo(parsed.children[0])?.rawBytes || der) as Uint8Array<ArrayBuffer>
  }
  const curve = algorithm.includes("384") ? "P-384" : algorithm.includes("521") ? "P-521" : "P-256"
  return crypto.subtle.importKey("spki", asArrayBuffer(spki), { name: "ECDSA", namedCurve: curve }, false, ["verify"])
}

async function getRootAnchor() {
  if (!rootAnchorPromise) {
    rootAnchorPromise = fetch(verificationUrl("/api/primary-key"), { credentials: "include", headers: { Accept: "application/json" } }).then(async (response) => {
      const payload = await response.json().catch(() => null) as { data?: RootTrustAnchor; message?: string } | null
      if (!response.ok || !payload?.data) throw new Error(payload?.message || `无法获取根信任锚点 (HTTP ${response.status})`)
      const anchor = payload.data
      if (!anchor.public_key_pem || !anchor.algorithm) throw new Error("根信任锚点响应缺少公钥或算法")
      return anchor
    }).catch((error) => {
      rootAnchorPromise = null
      throw error
    })
  }
  return rootAnchorPromise
}

async function getRootKey(anchor: RootTrustAnchor) {
  if (!rootCryptoKeyPromise) {
    rootCryptoKeyPromise = importPEMPublicKey(anchor.public_key_pem, anchor.algorithm).catch((error) => {
      rootCryptoKeyPromise = null
      throw error
    })
  }
  return rootCryptoKeyPromise
}

function step(progress: VerificationProgress | undefined, id: VerificationStepId, status: VerificationStepStatus, detail: string) {
  progress?.({ id, status, detail })
}

function requireRange(bytes: Uint8Array, start: number, length: number) {
  if (!Number.isSafeInteger(start) || !Number.isSafeInteger(length) || start < 0 || length < 0 || start + length > bytes.length) {
    throw new Error("PDF ByteRange 超出文件边界")
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

export async function verifyCredentialPdf(file: File, progress?: VerificationProgress): Promise<CredentialVerificationResult> {
  if (file.type && file.type !== "application/pdf") throw new Error("请选择 PDF 文件")
  const bytes = new Uint8Array(await file.arrayBuffer())
  if (bytes.length < 32) throw new Error("PDF 文件内容过短")
  const anchor = await getRootAnchor()

  step(progress, "parse", "running", "搜索 PDF 的 ByteRange 与签名容器...")
  const pdfText = decodeLatin1(bytes)
  if (!pdfText.startsWith("%PDF-")) throw new Error("文件不是有效的 PDF")
  const rangeMatch = /\/ByteRange\s*\[\s*(\d+)\s+(\d+)\s+(\d+)\s+(\d+)\s*\]/.exec(pdfText)
  if (!rangeMatch) throw new Error("未找到 /ByteRange，该文档未经过 PAdES 数字签名")
  const ranges = rangeMatch.slice(1).map(Number)
  requireRange(bytes, ranges[0], ranges[1])
  requireRange(bytes, ranges[2], ranges[3])
  const contentsMatch = /\/Contents\s*<([0-9A-Fa-f\s]+)>/.exec(pdfText)
  if (!contentsMatch) throw new Error("未找到 /Contents 签名容器")
  const signatureHex = contentsMatch[1].replace(/\s/g, "")
  if (!signatureHex || signatureHex.length % 2 !== 0) throw new Error("/Contents 签名容器不是有效的十六进制数据")
  const signatureContainer = Uint8Array.from((signatureHex.match(/.{2}/g) || []).map((value) => Number.parseInt(value, 16)))
  const p7Root = parseASN1(signatureContainer)
  step(progress, "parse", "success", `R1:[${ranges[0]}, ${ranges[1]}] R2:[${ranges[2]}, ${ranges[3]}]`)

  step(progress, "digest", "running", "在浏览器内存中计算正文摘要...")
  const contentBytes = new Uint8Array(ranges[1] + ranges[3])
  contentBytes.set(bytes.subarray(ranges[0], ranges[0] + ranges[1]), 0)
  contentBytes.set(bytes.subarray(ranges[2], ranges[2] + ranges[3]), ranges[1])

  const contentInfo = p7Root.children[1]
  const signedData = contentInfo?.children[0]
  if (!signedData) throw new Error("无效的 PKCS#7 SignedData 容器")
  const certificates = signedData.children.find((child) => child.tag === 0xa0)
  const signerInfos = signedData.children.find((child) => child.tag === 0x31)
  const leafCertificate = certificates?.children.find((child) => child.tag === 0x30)
  const signerInfo = signerInfos?.children[0]
  if (!leafCertificate || !signerInfo) throw new Error("PKCS#7 中缺少工作证书或 SignerInfo")
  const signerDigestAlgorithm = signerInfo.children.find((child) => child.tag === 0x30 && child.children[0]?.tag === 0x06)
  const authenticatedAttributes = signerInfo.children.find((child) => child.tag === 0xa0)
  const signature = signerInfo.children.find((child) => child.tag === 0x04)?.contentBytes
  if (!signerDigestAlgorithm || !authenticatedAttributes || !signature) throw new Error("SignerInfo 缺少摘要算法、属性集或签名")
  const digestOID = decodeOID(signerDigestAlgorithm.children[0].contentBytes)
  const hashName = hashNameForOID(digestOID)
  const computedDigest = hex(await crypto.subtle.digest(hashName, contentBytes))
  step(progress, "digest", "success", `${hashName}: ${computedDigest.slice(0, 16)}...`)

  step(progress, "chain", "running", "检查二级工作证书指纹与有效期...")
  const leafFingerprint = hex(await crypto.subtle.digest("SHA-256", asArrayBuffer(leafCertificate.rawBytes)))
  if ((anchor.revoked_leaf_fingerprints || []).some((item) => item.toLowerCase() === leafFingerprint.toLowerCase())) {
    throw new Error(`二级签名证书已被吊销 (指纹: ${leafFingerprint})`)
  }
  const leafTBS = leafCertificate.children[0]
  const leafSignatureAlgorithm = leafCertificate.children[1]
  const leafSignatureValue = leafCertificate.children[2]?.contentBytes.slice(1)
  const leafSPKI = findSubjectPublicKeyInfo(leafTBS)
  if (!leafTBS || !leafSignatureAlgorithm || !leafSignatureValue || !leafSPKI) throw new Error("X.509 工作证书结构无效")
  const curveOID = decodeOID(leafSPKI.children[0].children.find((child) => child.tag === 0x06)?.contentBytes || new Uint8Array())
  const curve = curveForOID(curveOID)
  const leafSignatureOID = decodeOID(leafSignatureAlgorithm.children[0].contentBytes)
  const leafSignatureHash = hashNameForOID(leafSignatureOID)
  const trusted = await crypto.subtle.verify(
    { name: "ECDSA", hash: leafSignatureHash },
    await getRootKey(anchor),
    derSignatureToP1363(leafSignatureValue, curve.size),
    asArrayBuffer(leafTBS.rawBytes),
  )
  if (!trusted) throw new Error("二级工作证书无法由官方根 CA 验证")
  const validity = parseCertificateTimes(leafTBS)
  const now = Date.now()
  if (validity.from && now < validity.from.getTime()) throw new Error("二级工作证书尚未生效")
  if (validity.to && now > validity.to.getTime()) throw new Error("二级工作证书已过期")
  step(progress, "chain", "success", `证书指纹: ${leafFingerprint.slice(0, 16)}... (信任链有效)`)

  step(progress, "certificate", "success", `${subjectCommonName(leafTBS)} / ${curve.name}`)
  step(progress, "signature", "running", "验证 PKCS#7 受保护属性集签名...")
  const signedAttributes = replaceImplicitSetTag(authenticatedAttributes)
  const leafKey = await crypto.subtle.importKey("spki", asArrayBuffer(leafSPKI.rawBytes), { name: "ECDSA", namedCurve: curve.name }, false, ["verify"])
  const signatureValid = await crypto.subtle.verify(
    { name: "ECDSA", hash: hashName },
    leafKey,
    derSignatureToP1363(signature, curve.size),
    asArrayBuffer(signedAttributes),
  )
  if (!signatureValid) throw new Error("PKCS#7 数字签名核验失败，正文或属性可能已被修改")

  let embeddedDigest = ""
  let ulid = ""
  for (const attribute of authenticatedAttributes.children) {
    if (attribute.children.length < 2 || attribute.children[0].tag !== 0x06) continue
    const oid = decodeOID(attribute.children[0].contentBytes)
    const value = attribute.children[1].children[0]
    if (!value) continue
    if (oid === "1.2.840.113549.1.9.4") embeddedDigest = hex(value.contentBytes)
    if (oid === "1.3.6.1.4.1.99999.1") ulid = readASN1String(value).toUpperCase()
  }
  if (embeddedDigest.toLowerCase() !== computedDigest.toLowerCase()) throw new Error("正文摘要与签名属性不一致，文件可能已被篡改")
  step(progress, "signature", "success", "签名与正文摘要完全一致")
  if (!ULID_PATTERN.test(ulid)) throw new Error("签名属性中未找到合法的 26 位证书 ULID")
  step(progress, "identity", "success", `证书主键: ${ulid}`)

  step(progress, "lifecycle", "running", "在线核验文件指纹与证书生命周期...")
  const pdfHash = hex(await crypto.subtle.digest("SHA-256", asArrayBuffer(bytes)))
  const response = await fetch(verificationUrl(`/api/check-validity?cred_ulid=${encodeURIComponent(ulid)}&pdf_hash=${encodeURIComponent(pdfHash)}`), { credentials: "include", headers: { Accept: "application/json" } })
  const payload = await response.json().catch(() => null) as { data?: LifecycleResponse; message?: string } | null
  if (!response.ok || !payload?.data) throw new Error(payload?.message || `在线生命周期查询失败 (HTTP ${response.status})`)
  const lifecycle = payload.data
  step(progress, "lifecycle", "success", `状态: ${String(lifecycle.status)}`)
  return {
    lifecycle,
    ulid,
    pdfHash,
    signerSubject: subjectCommonName(leafTBS),
    leafFingerprint,
    cryptoSuite: `${curve.name} / ${hashName}`,
    rootKeyId: anchor.key_id || "",
    leafValidFrom: validity.from,
    leafValidTo: validity.to,
  }
}
