# Port forwarding rules for http and https
:local destination 10.0.0.11;
/ip firewall nat
add action=dst-nat chain=dstnat dst-port=80  nth=5,1 in-interface-list=WAN protocol=tcp to-addresses=10.0.0.11  to-ports=80
add action=dst-nat chain=dstnat dst-port=80  nth=5,2 in-interface-list=WAN protocol=tcp to-addresses=10.0.0.12  to-ports=80
add action=dst-nat chain=dstnat dst-port=80  nth=5,3 in-interface-list=WAN protocol=tcp to-addresses=10.0.0.201 to-ports=80
add action=dst-nat chain=dstnat dst-port=80  nth=5,4 in-interface-list=WAN protocol=tcp to-addresses=10.0.0.202 to-ports=80
add action=dst-nat chain=dstnat dst-port=80  nth=5,5 in-interface-list=WAN protocol=tcp to-addresses=10.0.0.203 to-ports=80

/ip firewall nat
add action=dst-nat chain=dstnat dst-port=443 nth=5,1 in-interface-list=WAN protocol=tcp to-addresses=10.0.0.11  to-ports=443
add action=dst-nat chain=dstnat dst-port=443 nth=5,2 in-interface-list=WAN protocol=tcp to-addresses=10.0.0.12  to-ports=443
add action=dst-nat chain=dstnat dst-port=443 nth=5,3 in-interface-list=WAN protocol=tcp to-addresses=10.0.0.201 to-ports=443
add action=dst-nat chain=dstnat dst-port=443 nth=5,4 in-interface-list=WAN protocol=tcp to-addresses=10.0.0.202 to-ports=443
add action=dst-nat chain=dstnat dst-port=443 nth=5,5 in-interface-list=WAN protocol=tcp to-addresses=10.0.0.203 to-ports=443

# Drop all port-forwarded packets that aren't coming from cloudflare.
:foreach port in={80; 443} do={
	/ip firewall filter add          \
		action=drop                  \
		chain=forward                \
		connection-nat-state=dstnat  \
		dst-port=$port               \
		protocol=tcp                 \
		src-address-list=!cloudflare \
		in-interface-list=WAN        \
		log=yes                      \
		log-prefix=drop-no-cf        \
		comment="Drop dstnat packets not from cloudflare: $port";
}