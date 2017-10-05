# AdGuard DNS (beta)
[![](https://travis-ci.org/AdguardTeam/AdguardDNS.svg?branch=master)](https://travis-ci.org/AdguardTeam/AdguardDNS)

> **Our IP addresses** 
>
> Default mode: 
> *176.103.130.130, 
> 176.103.130.131*
>
>Family protection:
>*176.103.130.132, 
>176.103.130.134*

### What is AdGuard DNS?

AdGuard DNS is an alternative way to block ads, trackers and phishing websites, and also a parental control instrument.

### How does AdGuard DNS work?

If you configure your network to use our DNS servers, every time a website sends an ad or phishing request, our server sends back a null response. AdGuard has a database of domain names that serve for ad, tracking or phishing purposes, and this database is regularly updated.

AdGuard DNS works in two modes:

 - 'Default' mode blocks ads, various trackers and malware & phishing websites. 
 - 'Family protection' does the same, but also blocks websites with adult content.

This repository contains filters used by AdGuard DNS server.

#### DNSCrypt

AdGuard supports DNSCrypt — a special protocol that encrypts communication with the DNS server, thus preventing tampering and tracking by any third party, including your ISP. Read more about DNSCrypt [here](https://dnscrypt.org/).

### How to use AdGuard DNS?

The detailed guide can be found on [our website](https://adguard.com/en/adguard-dns/instruction.html#instruction).

If you come across any problem, or have a suggestion, head to [this page](https://github.com/AdguardTeam/AdguardDNS/issues) and click on the *New issue* button.

### What's next?

We plan to make AdGuard DNS open source in the near future.
