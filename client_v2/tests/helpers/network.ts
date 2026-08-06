import { networkInterfaces } from 'os';
import type { NetworkInterfaceInfo } from 'node:os';

interface DHCPConfig {
    interfaceName: string;
    rangeStart: string;
    rangeEnd: string;
    subnetMask: string;
}

const DEFAULT_SUBNET_MASK = '255.255.255.0';
const DEFAULT_SUBNET_MASK_OCTETS = DEFAULT_SUBNET_MASK.split('.').map(Number);

function checkIsIPv4(addr: NetworkInterfaceInfo): boolean {
    return addr.family === 'IPv4' && !addr.internal;
}

function calculateNetwork(ip: number[], mask: number[]): number[] {
    // Calculate the network address by applying the bitwise AND operation.
    // eslint-disable-next-line no-bitwise
    return ip.map((octet, i) => octet & mask[i]);
}

function calculateBroadcast(network: number[], mask: number[]): number[] {
    // Calculate the broadcast address by ORing the network address with the inverted mask.
    // eslint-disable-next-line no-bitwise
    return network.map((octet, i) => octet | (~mask[i] & 255));
}

export function getDHCPConfig(): DHCPConfig {
    const interfaces = networkInterfaces();

    // Select the first interface that has a valid non-internal IPv4 address.
    const ipV4Interface = Object.entries(interfaces)
        .map(([name, addresses]) => ({ name, addresses }))
        .find((i) => i.addresses?.some(checkIsIPv4));

    if (!ipV4Interface) {
        throw new Error('No suitable network interface found');
    }

    // Get the first valid IPv4 address from the interface.
    const ipv4Address = ipV4Interface.addresses.find(checkIsIPv4);

    const ip = ipv4Address.address.split('.').map(Number);
    const mask = ipv4Address.netmask?.split('.').map(Number) || DEFAULT_SUBNET_MASK_OCTETS;

    const network = calculateNetwork(ip, mask);

    // Calculate first usable address (network address + 1)
    const rangeStart = [...network];
    rangeStart[3] = network[3] + 1;

    // Calculate broadcast address and then the last usable address (broadcast - 1)
    const broadcast = calculateBroadcast(network, mask);
    const rangeEnd = [...broadcast];
    rangeEnd[3] = broadcast[3] - 1;

    return {
        interfaceName: ipV4Interface.name,
        rangeStart: rangeStart.join('.'),
        rangeEnd: rangeEnd.join('.'),
        subnetMask: ipv4Address.netmask || DEFAULT_SUBNET_MASK,
    };
}
