import secrets
import hashlib
import base64


def generate_verifier() -> str:
	return secrets.token_urlsafe(96)


def create_challenge(verifier: str) -> str:
	# https://datatracker.ietf.org/doc/html/rfc7636#section-4.2
	# S256:
    #   code_challenge = BASE64URL-ENCODE(SHA256(ASCII(code_verifier)))
	#
	vb = verifier.encode("ascii")
	hashed = hashlib.sha256(vb).digest()
	encoded = base64.urlsafe_b64encode(hashed)
	# convert from bytes to str and remove '='
	return encoded.decode("ascii").rstrip("=")


def verify(verifier: str, challenge: str):
	computed = create_challenge(verifier)
	if computed != challenge:
		raise ValueError(f"verifier does not match the challenge")
