export const hasMinLength = (v: string) => v.length >= 8;

export const hasLowercase = (v: string) => /\p{Ll}/u.test(v);

export const hasUppercase = (v: string) => /\p{Lu}/u.test(v);

/** Any printable character: control characters are rejected, emoji/ZWJ allowed. */
export const hasAllowedChars = (v: string) => !/\p{Cc}/u.test(v);

/** A digit, punctuation, or symbol — emoji count as special; letters and spaces do not. */
export const hasNumberOrSpecial = (v: string) => /[\p{N}\p{P}\p{S}]/u.test(v);

export const stripZoneId = (ip: string) => ip.split('%')[0];

export const getDnsAddressWithPort = (ip: string, port: number) => {
    const normalizedIp = stripZoneId(ip);

    if (normalizedIp.includes(':') && !normalizedIp.includes('[')) {
        return `[${normalizedIp}]:${port}`;
    }

    return `${normalizedIp}:${port}`;
};
