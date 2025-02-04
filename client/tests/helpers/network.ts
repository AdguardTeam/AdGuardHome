import { networkInterfaces } from 'os';
import type { NetworkInterfaceInfo } from 'node:os';

interface DHCPConfig {
    interfaceName: string;
    rangeStart: string;
    rangeEnd: string;
    subnetMask: string;
}

const DEFAULT_SUBNET_MASK = '255.255.255.0';

function checkIsIPv4(addr: NetworkInterfaceInfo): boolean {
    return addr.family === 'IPv4' && !addr.internal;
}

function calculateNetwork(ip: number[], mask: number[]): number[] {
    // eslint-disable-next-line no-bitwise
    return ip.map((octet, i) => octet & mask[i]);
}

function calculateBroadcast(network: number[], mask: number[]): number[] {
    // eslint-disable-next-line no-bitwise
    return network.map((octet, i) => octet | (~mask[i] & 255));
}

export function getDHCPConfig(): DHCPConfig {
    const interfaces = networkInterfaces();

    const ipV4Interface = Object.entries(interfaces)
        .map(([name, addresses]) => ({
            name,
            addresses,
        }))
        .find((i) => {
            return i.addresses?.some((addr) => checkIsIPv4(addr));
        });

    if (!ipV4Interface) {
        throw new Error('No suitable network interface found');
    }

    const ipv4Address = ipV4Interface.addresses.find((addr) => checkIsIPv4(addr));

    const ip = ipv4Address.address.split('.').map(Number);
    const mask = ipv4Address.netmask?.split('.').map(Number) || DEFAULT_SUBNET_MASK.split('.');

    const network = calculateNetwork(ip, mask);

    // Calculate first and last usable addresses (excluding network and broadcast)
    const rangeStart = [...network];
    rangeStart[3] = network[3] + 1;

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
