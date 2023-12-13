
# Port forwarding rules for http and https
# /ip firewall nat
# add action=dst-nat chain=dstnat dst-port=80  nth=5,1 in-interface-list=WAN protocol=tcp to-addresses=10.0.0.11  to-ports=80
# add action=dst-nat chain=dstnat dst-port=80  nth=5,2 in-interface-list=WAN protocol=tcp to-addresses=10.0.0.12  to-ports=80
# add action=dst-nat chain=dstnat dst-port=80  nth=5,3 in-interface-list=WAN protocol=tcp to-addresses=10.0.0.21 to-ports=80
# add action=dst-nat chain=dstnat dst-port=80  nth=5,4 in-interface-list=WAN protocol=tcp to-addresses=10.0.0.22 to-ports=80
# add action=dst-nat chain=dstnat dst-port=80  nth=5,5 in-interface-list=WAN protocol=tcp to-addresses=10.0.0.23 to-ports=80
#
# /ip firewall nat
# add action=dst-nat chain=dstnat dst-port=443 nth=5,1 in-interface-list=WAN protocol=tcp to-addresses=10.0.0.11  to-ports=443
# add action=dst-nat chain=dstnat dst-port=443 nth=5,2 in-interface-list=WAN protocol=tcp to-addresses=10.0.0.12  to-ports=443
# add action=dst-nat chain=dstnat dst-port=443 nth=5,3 in-interface-list=WAN protocol=tcp to-addresses=10.0.0.21 to-ports=443
# add action=dst-nat chain=dstnat dst-port=443 nth=5,4 in-interface-list=WAN protocol=tcp to-addresses=10.0.0.22 to-ports=443
# add action=dst-nat chain=dstnat dst-port=443 nth=5,5 in-interface-list=WAN protocol=tcp to-addresses=10.0.0.23 to-ports=443

:foreach port in={80; 443} do={
    :local i 1;
    :foreach address in={
        11;
        12;
        21;
        22;
        23;
    } do={
        :put "10.0.0.$address:$port 5,$i";
        # /ip firewall nat add             \
        #     action=dst-nat               \
        #     chain=dstnat                 \
        #     dst-port=$port               \
        #     nth=5,$i                     \
        #     in-interface-list=WAN        \
        #     protocol=tcp                 \
        #     to-address="10.0.0.$address" \
        #     to-port=$port                \
        #     comment=""
        :set i ($i+1);
    }
};

# Drop all port-forwarded packets that aren't coming from cloudflare.
#:foreach port in={80; 443} do={
#	/ip firewall filter add          \
#		action=drop                  \
#		chain=forward                \
#		connection-nat-state=dstnat  \
#		dst-port=$port               \
#		protocol=tcp                 \
#		src-address-list=!cloudflare \
#		in-interface-list=WAN        \
#		log=yes                      \
#		log-prefix=drop-no-cf        \
#		comment="Drop dstnat packets not from cloudflare: $port";
#};

#/ip firewall nat
#add action=masquerade chain=srcnat comment="defconf: masquerade" ipsec-policy=out,none out-interface-list=WAN
#add action=dst-nat chain=dstnat dst-port=80 in-interface-list=WAN protocol=tcp to-addresses=10.0.0.11 to-ports=80
#add action=dst-nat chain=dstnat dst-port=443 in-interface-list=WAN protocol=tcp to-addresses=10.0.0.11 to-ports=443
