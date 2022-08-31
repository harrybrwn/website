import secrets
import hashlib
import base64
import re

_plus = re.compile('\+')
_slash = re.compile('\/')
_last_equals = re.compile('=+$')

def b64(b: bytes) -> str:
	enc = base64.urlsafe_b64encode(b)
	enc = enc.decode("ascii")
	# https://www.rfc-editor.org/rfc/rfc4648#section-5
	enc = _plus.sub("-", enc)
	enc = _slash.sub("_", enc)
	return _last_equals.sub("", enc)



def generate_verifier() -> str:
	# token_urlsafe generates random a base64 encoded string.  We want a
	# verifier that is 128 characters long so we use 96 to account for the
	# base64 encoding.
	return secrets.token_urlsafe(96)


def create_challenge(verifier: str) -> str:
	# https://datatracker.ietf.org/doc/html/rfc7636#section-4.2
	# https://www.rfc-editor.org/rfc/rfc7636#section-4.2
	# S256:
    #   code_challenge = BASE64URL-ENCODE(SHA256(ASCII(code_verifier)))
	#
	vb = verifier.encode("ascii")
	hashed = hashlib.sha256(vb).digest()
	return b64(hashed)


def verify(verifier: str, challenge: str):
	computed = create_challenge(verifier)
	if computed != challenge:
		raise ValueError(f"verifier does not match the challenge")
