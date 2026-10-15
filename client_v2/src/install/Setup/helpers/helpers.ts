export const hasMinLength = (v: string) => v.length >= 8;

export const hasLowercase = (v: string) => /\p{Ll}/u.test(v);

export const hasUppercase = (v: string) => /\p{Lu}/u.test(v);

export const stripZoneId = (ip: string) => ip.split('%')[0];

export const getDnsAddressWithPort = (ip: string, port: number) => {
    const normalizedIp = stripZoneId(ip);

    if (normalizedIp.includes(':') && !normalizedIp.includes('[')) {
        return `[${normalizedIp}]:${port}`;
    }

    return `${normalizedIp}:${port}`;
};
