export const hasMinLength = (v: string) => v.length >= 8;

export const hasLowercase = (v: string) => /[a-z]/.test(v);

export const hasUppercase = (v: string) => /[A-Z]/.test(v);

export const hasAllowedAsciiOnly = (v: string) => /^[\x20-\x7E]*$/.test(v);

export const hasNumberOrSpecial = (v: string) => /[\d\W_]/.test(v);

export const stripZoneId = (ip: string) => ip.split('%')[0];
