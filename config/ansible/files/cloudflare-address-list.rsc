# Download cloudflare's address list
/tool fetch "https://www.cloudflare.com/ips-v4" dst-path="disk1/cloudflare-ipv4.txt";
:local content [ /file get "disk1/cloudflare-ipv4.txt" contents ];
# /file remove "disk1/cloudflare-ipv4.txt";

:local flen [ :len $content ];
:local lineEnd 0;
:local prevEnd 0;
:local line "";

# Loop over the contents of the file and add the addresses
:do {
	:set lineEnd [ :find $content "\n" $prevEnd ];
	:set line [ :pick $content $prevEnd $lineEnd ];
	:set prevEnd ( $lineEnd + 1 );
	:local entry [ :pick $line 0 $lineEnd ];
	:if ([:len $entry]<=1) do={
		# Stop the loop if the line has one character or less
		:set lineEnd $flen;
	} else={
		:if ([ :len [ /ip firewall address-list find where list=cloudflare and address=$entry ] ]=0) do={
			/ip firewall address-list add list=cloudflare address=$entry;
		} else={
			:put "$entry already in the address list";
		}
	}
} while ($lineEnd < $flen);
