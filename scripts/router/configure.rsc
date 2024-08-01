#| Welcome to RouterOS!
#|    1) Set a strong router password in the System > Users menu
#|    2) Upgrade the software in the System > Packages menu
#|    3) Enable firewall on untrusted networks
#| -----------------------------------------------------------------------------
#| RouterMode:
#|  * WAN port is protected by firewall and enabled DHCP client
#|  * Ethernet interfaces (except WAN port/s) are part of LAN bridge
#| LAN Configuration:
#|     IP address 10.0.0.1/24 is set on bridge (LAN port)
#|     DHCP Server: enabled;
#|     DNS: enabled;
#| WAN (gateway) Configuration:
#|     gateway:  sfp1 ;
#|     ip4 firewall:  enabled;
#|     NAT:   enabled;
#|     DHCP Client: enabled;

# wait for interfaces
:local count 0;
:while ([/interface ethernet find] = "") do={
  :if ($count = 30) do={
    :log warning "DefConf: Unable to find ethernet interfaces";
    /quit;
  }
  :delay 1s; :set count ($count +1);
};

/interface list add name=WAN comment="defconf"
/interface list add name=LAN comment="defconf"
/interface bridge
  add name=bridge disabled=no auto-mac=yes protocol-mode=rstp comment="defconf";
:local bMACIsSet 0;
:foreach k in=[/interface find where !(slave=yes  || name="sfp1" || name~"bridge")] do={
  :local tmpPortName [/interface get $k name];
  :if ($bMACIsSet = 0) do={
    :if ([/interface get $k type] = "ether") do={
      /interface bridge set "bridge" auto-mac=no admin-mac=[/interface get $tmpPortName mac-address];
      :set bMACIsSet 1;
    }
  }
  :if (([/interface get $k type] != "ppp-out") && ([/interface get $k type] != "lte")) do={
    /interface bridge port
      add bridge=bridge interface=$tmpPortName comment=defconf;
  }
}
/ip pool add name="default-dhcp" ranges=10.0.0.50-10.0.0.254;
/ip dhcp-server
  add name=defconf address-pool="default-dhcp" interface=bridge lease-time=30m disabled=no;
/ip dhcp-server network add address=10.0.0.0/24 gateway=10.0.0.1 comment="defconf";
/ip address add address=10.0.0.1/24 interface=bridge comment="defconf";
/ip dns {
  set allow-remote-requests=yes
  static add name=router.lan address=10.0.0.1 comment=defconf
  static add name=rpi1.lan address=10.0.0.21
  static add name=rpi2.lan address=10.0.0.22
  static add name=rpi3.lan address=10.0.0.23
}
/ip dhcp-server {
  lease add address=10.0.0.2  mac-address=94:A6:7E:B1:F4:13 server=defconf
  lease add address=10.0.0.11 mac-address=00:24:9B:29:EF:11 server=defconf
  lease add address=10.0.0.13 mac-address=EC:A8:6B:38:07:5F server=defconf
  lease add address=10.0.0.21 mac-address=DC:A6:32:A7:40:F4 server=defconf
  lease add address=10.0.0.22 mac-address=DC:A6:32:A7:40:91 server=defconf
  lease add address=10.0.0.23 mac-address=DC:A6:32:A6:D0:5F server=defconf
  lease add address=10.0.0.8  mac-address=00:E0:4C:68:00:EC server=defconf
}

# Cloudflare address list
/ip firewall {
  address-list add list=cloudflare address=173.245.48.0/20
  address-list add list=cloudflare address=103.21.244.0/22
  address-list add list=cloudflare address=103.22.200.0/22
  address-list add list=cloudflare address=103.31.4.0/22
  address-list add list=cloudflare address=141.101.64.0/18
  address-list add list=cloudflare address=108.162.192.0/18
  address-list add list=cloudflare address=190.93.240.0/20
  address-list add list=cloudflare address=188.114.96.0/20
  address-list add list=cloudflare address=197.234.240.0/22
  address-list add list=cloudflare address=198.41.128.0/17
  address-list add list=cloudflare address=162.158.0.0/15
  address-list add list=cloudflare address=104.16.0.0/13
  address-list add list=cloudflare address=104.24.0.0/14
  address-list add list=cloudflare address=172.64.0.0/13
  address-list add list=cloudflare address=131.0.72.0/22
}
/ip firewall {
  address-list add list=not_in_internet address=0.0.0.0/8 comment=RFC6890
  address-list add list=not_in_internet address=172.16.0.0/12 comment=RFC6890
  address-list add list=not_in_internet address=192.168.0.0/16 comment=RFC6890
  address-list add list=not_in_internet address=10.0.0.0/8 comment=RFC6890
  address-list add list=not_in_internet address=169.254.0.0/16 comment=RFC6890
  address-list add list=not_in_internet address=127.0.0.0/8 comment=RFC6890
  address-list add list=not_in_internet address=224.0.0.0/4 comment=Multicast
  address-list add list=not_in_internet address=198.18.0.0/15 comment=RFC6890
  address-list add list=not_in_internet address=192.0.0.0/24 comment=RFC6890
  address-list add list=not_in_internet address=192.0.2.0/24 comment=RFC6890
  address-list add list=not_in_internet address=198.51.100.0/24 comment=RFC6890
  address-list add list=not_in_internet address=203.0.113.0/24 comment=RFC6890
  address-list add list=not_in_internet address=100.64.0.0/10 comment=RFC6890
  address-list add list=not_in_internet address=240.0.0.0/4 comment=RFC6890
  address-list add list=not_in_internet address=192.88.99.0/24 comment="6to4 relay Anycast [RFC 3068]"
}

/interface list member add list=LAN interface=bridge comment="defconf"
/interface list member add list=WAN interface=sfp1 comment="defconf"
/ip firewall nat add chain=srcnat out-interface-list=WAN ipsec-policy=out,none action=masquerade comment="defconf: masquerade"
/ip firewall {
  nat add action=dst-nat chain=dstnat dst-port=80  in-interface-list=WAN protocol=tcp to-addresses=10.0.0.11 to-ports=80
  nat add action=dst-nat chain=dstnat dst-port=443 in-interface-list=WAN protocol=tcp to-addresses=10.0.0.11 to-ports=443
}
/ip firewall {
  filter add chain=input action=accept connection-state=established,related,untracked comment="defconf: accept established,related,untracked"
  filter add chain=input action=drop connection-state=invalid comment="defconf: drop invalid"
  filter add chain=input action=drop protocol=icmp in-interface-list=!LAN log=no log-prefix="PING" comment="defconf: drop ICMP"
  filter add chain=input action=accept dst-address=127.0.0.1 comment="defconf: accept to local loopback (for CAPsMAN)"
  filter add chain=input action=drop in-interface-list=!LAN comment="defconf: drop all not coming from LAN"
  filter add chain=forward action=accept ipsec-policy=in,ipsec comment="defconf: accept in ipsec policy"
  filter add chain=forward action=accept ipsec-policy=out,ipsec comment="defconf: accept out ipsec policy"
  filter add chain=forward action=fasttrack-connection connection-state=established,related comment="defconf: fasttrack"
  filter add chain=forward action=accept connection-state=established,related,untracked comment="defconf: accept established,related, untracked"
  filter add chain=forward action=drop connection-state=invalid comment="defconf: drop invalid"
  filter add chain=forward action=drop connection-state=new connection-nat-state=!dstnat in-interface-list=WAN comment="defconf: drop all from WAN not DSTNATed"
  # Block all non-cloudflare IPs that are forwarded in the NAT
  filter add chain=forward action=drop connection-nat-state=dstnat dst-port=80  src-address-list=!cloudflare protocol=tcp in-interface-list=WAN log=yes log-prefix=NOT_CF comment="Drop dstnat packets not from cloudflare: port 80"
  filter add chain=forward action=drop connection-nat-state=dstnat dst-port=443 src-address-list=!cloudflare protocol=tcp in-interface-list=WAN log=yes log-prefix=NOT_CF comment="Drop dstnat packets not from cloudflare: port 443"
}
/ip neighbor discovery-settings set discover-interface-list=LAN
/tool mac-server set allowed-interface-list=LAN
/tool mac-server mac-winbox set allowed-interface-list=LAN

# Add my ssh key
/user ssh-keys import user=admin public-key-file=disk1/mikrotik_hexs_routerboard.pub

# Disable most services
/ip service disable telnet,ftp,www,www-ssl,api-ssl

# TODO See https://wiki.mikrotik.com/wiki/Manual:Securing_Your_Router
#      Use '/tool mac-server sessions' to enable the mac server/discovery only
#      for the mac-address currently configuring the router.
#
#/tool mac-server set allowed-interface-list=none
#/tool mac-server mac-winbox set allowed-interface-list=none
#/tool mac-server ping set enabled=no
#/ip neighbor discovery-settings set discover-interface-list=none
#/tool bandwidth-server set enabled=no

/system logging add topics="info"    action="remote"
/system logging add topics="warning" action="remote"
/system logging add topics="error"   action="remote"
/system logging add topics="system"  action="remote"
/system logging add topics="account" action="remote"

/system logging add topics="firewall"    action="remote"
/system logging add topics="interface"   action="remote"
/system logging add topics="smb"         action="remote"
/system logging add topics="certificate" action="remote"
/system logging add topics="manager"     action="remote"
/system logging add topics="ssh,!debug"  action="remote"
/system logging add topics="dhcp,!debug" action="remote"

# Enable the dhcp client
/ip dhcp-client add interface=sfp1 disabled=no comment="defconf";

# vim: ts=2 sts=2 sw=2
