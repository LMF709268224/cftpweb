// Numeric values match gcreds.CredentialSource in the JSON API.
export const CREDENTIAL_SOURCE_APPLICATION = 1
export const CREDENTIAL_SOURCE_PDF_CERT = 2

export type CredentialSource = typeof CREDENTIAL_SOURCE_APPLICATION | typeof CREDENTIAL_SOURCE_PDF_CERT
