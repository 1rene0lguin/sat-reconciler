package apperrors

// === Authentication Errors ===
const (
	ErrAuthBuild   = "error building auth request"
	ErrAuthSign    = "error signing auth timestamp"
	ErrAuthRequest = "error sending auth request"
	ErrAuthParse   = "error reading auth token"
	ErrEmptyToken  = "authentication token is empty"
	ErrAuthFailed  = "authentication failed"
	ErrAuthHTTP    = "HTTP error during authentication"
)

// === Key & Certificate Errors ===
const (
	ErrReadingKey         = "error reading private key"
	ErrReadingCert        = "error reading certificate"
	ErrDecryptKey         = "incorrect password or key decryption failed"
	ErrUnsupportedKeyFmt  = "unsupported key format or incorrect password"
	ErrKeyNotRSA          = "private key is not RSA"
	ErrEncryptedKeyNotRSA = "encrypted key is not RSA"
)

// === Request Building Errors ===
const (
	ErrBuilderInit    = "builder initialization error"
	ErrBuildRequest   = "error building request"
	ErrSignRequest    = "error signing request"
	ErrSignRSA        = "error signing RSA"
	ErrSignAuthRSA    = "error signing auth RSA"
	ErrRenderTemplate = "error rendering template"
)

// === HTTP / Network Errors ===
const (
	ErrRequestCreation = "request creation error"
	ErrNetworkError    = "network error"
	ErrHTTPError       = "HTTP error"
	ErrRateLimit       = "rate limit error"
	ErrRetryExhausted  = "request failed after retries"
	ErrRetryBodyReprod = "failed to reproduce request body for retry"
)

// === SAT Response Errors ===
const (
	ErrSATError          = "SAT error"
	ErrReadBody          = "reading body error"
	ErrReadResponse      = "reading response error"
	ErrXMLParsing        = "XML parsing error"
	ErrXMLUnrecognizable = "unrecognizable XML response structure"
	ErrEmptyUUID         = "empty UUID in SAT response"
	ErrXMLGeneration     = "XML generation error"
)

// === Parser Errors ===
const (
	ErrMalformedLine = "malformed line"
)

// === Zip / Package Errors ===
const (
	ErrZipEntry   = "error creating zip entry"
	ErrZipContent = "error writing zip content"
	ErrZipClose   = "error closing zip writer"
)
