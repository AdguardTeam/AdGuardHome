import { networkInterfaces } from 'os';

interface DHCPConfig {
    interfaceName: string;
    rangeStart: string;
    rangeEnd: string;
    subnetMask: string;
}

export function getDHCPConfig(): DHCPConfig {
    const interfaces = networkInterfaces();
    for (const [name, addresses] of Object.entries(interfaces)) {
        const ipv4Address = addresses?.find((addr) => addr.family === 'IPv4' && !addr.internal);
        if (ipv4Address) {
            const ip = ipv4Address.address.split('.').map(Number);
            const mask = ipv4Address.netmask?.split('.').map(Number) || [255, 255, 255, 0];
            
            // Calculate network address
            const network = ip.map((octet, i) => octet & mask[i]);
            
            // Calculate first and last usable addresses (excluding network and broadcast)
            const rangeStart = [...network];
            rangeStart[3] = network[3] + 1;
            
            const broadcast = network.map((octet, i) => octet | (~mask[i] & 255));
            const rangeEnd = [...broadcast];
            rangeEnd[3] = broadcast[3] - 1;

            return {
                interfaceName: name,
                rangeStart: rangeStart.join('.'),
                rangeEnd: rangeEnd.join('.'),
                subnetMask: ipv4Address.netmask || '255.255.255.0',
            };
        }
    }
    throw new Error('No suitable network interface found');
}
