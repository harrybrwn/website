import requests
import pkce


def main():
	s = requests.Session()
	verifier = pkce.generate_verifier()
	code_challenge = pkce.create_challenge(verifier)
	res = s.get(
		"https://auth.hrry.local/oauth2/auth",
		params={
			"client_id": "testid",
			"response_type": "code",
			"scope": "openid offline",
			"state": "abcdefghijklmnopqrstuv",
			"code_challenge": code_challenge,
			"code_challenge_method": "S256",
		},
		allow_redirects=False,
		verify=False,
	)
	print(res)
	print(res.headers)
	print(res.next.url)


if __name__ == '__main__':
	main()