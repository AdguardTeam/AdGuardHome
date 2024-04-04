package configmigrate

import "github.com/AdguardTeam/golibs/errors"

// migrateTo7 performs the following changes:
//
//	# BEFORE:
//	'schema_version': 6
//	'dhcp':
//	  'enabled': false
//	  'interface_name': vboxnet0
//	  'gateway_ip': '192.168.56.1'
//	  'subnet_mask': '255.255.255.0'
//	  'range_start': '192.168.56.10'
//	  'range_end': '192.168.56.240'
//	  'lease_duration': 86400
//	  'icmp_timeout_msec': 1000
//	# …
//
//	# AFTER:
//	'schema_version': 7
//	'dhcp':
//	  'enabled': false
//	  'interface_name': vboxnet0
//	  'dhcpv4':
//	    'gateway_ip': '192.168.56.1'
//	    'subnet_mask': '255.255.255.0'
//	    'range_start': '192.168.56.10'
//	    'range_end': '192.168.56.240'
//	    'lease_duration': 86400
//	    'icmp_timeout_msec': 1000
//	# …
func migrateTo7(diskConf yobj) (err error) {
	diskConf["schema_version"] = 7

	dhcp, ok, _ := fieldVal[yobj](diskConf, "dhcp")
	if !ok {
		return nil
	}

	dhcpv4 := yobj{}
	err = errors.Join(
		moveSameVal[string](dhcp, dhcpv4, "gateway_ip"),
		moveSameVal[string](dhcp, dhcpv4, "subnet_mask"),
		moveSameVal[string](dhcp, dhcpv4, "range_start"),
		moveSameVal[string](dhcp, dhcpv4, "range_end"),
		moveSameVal[int](dhcp, dhcpv4, "lease_duration"),
		moveSameVal[int](dhcp, dhcpv4, "icmp_timeout_msec"),
	)
	if err != nil {
		return err
	}

	dhcp["dhcpv4"] = dhcpv4

	return nil
}
