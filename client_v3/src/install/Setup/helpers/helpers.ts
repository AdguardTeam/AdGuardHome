export const hasMinLength = (v: string) => v.length >= 8;

export const hasLowercase = (v: string) => /[a-z]/.test(v);

export const hasUppercase = (v: string) => /[A-Z]/.test(v);

export const hasAllowedAsciiOnly = (v: string) => /^[\x20-\x7E]*$/.test(v);

export const hasNumberOrSpecial = (v: string) => /[\d\W_]/.test(v);

export const stripZoneId = (ip: string) => ip.split('%')[0];

export const getDnsAddressWithPort = (ip: string, port: number) => {
    const normalizedIp = stripZoneId(ip);

    if (normalizedIp.includes(':') && !normalizedIp.includes('[')) {
        return `[${normalizedIp}]:${port}`;
    }

    return `${normalizedIp}:${port}`;
};
