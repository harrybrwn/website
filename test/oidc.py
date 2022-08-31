import requests
import pkce
import urllib.parse


class Config:
	def __init__(self, auth_endpoint, login_endpoint, consent_endpoint):
		self.auth_endpoint = auth_endpoint
		self.login_endpoint = login_endpoint
		self.consent_endpoint = consent_endpoint


def main():
	v = "KHj1iHdNAAgozRF19uYwpof-nF7OUyZZBfRCpTzMfdrNyQVODwKJywzZR51ZkmLLGB9dy9Yz18Dc8PSi7QBx6t4idG-kJE4YQqcbH0NxuLqGiKcFeVAjFIGJSqp_ae7z"
	c = pkce.create_challenge(v)
	print(v)
	print(c)
	print(len(v), len(c))
	return
	s = requests.Session()
	verifier = pkce.generate_verifier()

	code_challenge = pkce.create_challenge(verifier)
	url = 'https://auth.hrry.local/oauth2/auth'
	config = Config(
		auth_endpoint="https://auth.hrry.local/oauth2/auth",
		login_endpoint="https://hrry.local/api/login",
		consent_endpoint="https://hrry.local/api/consent",
	)

	params = {
		"client_id": "testid",
		"response_type": "code",
		"scope": "openid offline",
		"state": "abcdefghijklmnopqrstuv",
		"code_challenge": code_challenge,
		"code_challenge_method": "S256",
	}
	# print(f"{url}?{urllib.parse.urlencode(params)}")
	res = s.get(
		"https://auth.hrry.local/oauth2/auth",
		params=params,
		allow_redirects=False,
		verify=False,
	)
	#print(res)
	#print(res.headers)
	#print(res.next.url)



if __name__ == '__main__':
	main()
