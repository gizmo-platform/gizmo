interface=wlan0
driver=nl80211
hw_mode=g
ssid={{.NetSSID}}
channel={{.Channel}}
auth_algs=1
wpa=2
wpa_key_mgmt=WPA-PSK
wpa_pairwise=TKIP
rsn_pairwise=CCMP
wpa_passphrase={{.NetPSK}}
macaddr_acl=0
ignore_broadcast_ssid=1
ctrl_interface=/var/run/hostapd
bridge=br0
