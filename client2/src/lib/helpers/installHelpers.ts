export enum NETWORK_TYPE {
    LOCAL = 'LOCAL',
    ETHERNET = 'ETHERNET',
    OTHER = 'OTHER',
}

export const chechNetworkType = (network: string | undefined) => {
    if (!network) {
        return NETWORK_TYPE.OTHER;
    }
    if (network.includes('en')) {
        return NETWORK_TYPE.ETHERNET;
    }
    if (network.includes('lo')) {
        return NETWORK_TYPE.LOCAL;
    }
};
