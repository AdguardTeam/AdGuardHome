# DHCP server

Contents:
* [Test setup with Virtual Box](#vbox)

<a id="vbox"></a>
## Test setup with Virtual Box

To set up a test environment for DHCP server you need:

* Linux host machine
* Virtual Box
* Virtual machine (guest OS doesn't matter)

### Configure client

1. Install Virtual Box and run the following command to create a Host-Only network:

        $ VBoxManage hostonlyif create

    You can check its status by `ip a` command.

    You can also set up Host-Only network using Virtual Box menu:

        File -> Host Network Manager...

2. Create your virtual machine and set up its network:

        VM Settings -> Network -> Host-only Adapter

3. Start your VM, install an OS.  Configure your network interface to use DHCP and the OS should ask for a IP address from our DHCP server.

### Configure server

1. Edit server configuration file 'AdGuardHome.yaml', for example:

        dhcp:
          enabled: true
          interface_name: vboxnet0
          gateway_ip: 192.168.56.1
          subnet_mask: 255.255.255.0
          range_start: 192.168.56.2
          range_end: 192.168.56.2
          lease_duration: 86400
          icmp_timeout_msec: 1000

2. Start the server

        ./AdGuardHome

    There should be a message in log which shows that DHCP server is ready:

        [info] DHCP: listening on 0.0.0.0:67
