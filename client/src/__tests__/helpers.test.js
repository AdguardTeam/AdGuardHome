import { getIpMatchListStatus, sortIp } from '../helpers/helpers';
import { IP_MATCH_LIST_STATUS } from '../helpers/constants';

describe('getIpMatchListStatus', () => {
    describe('IPv4', () => {
        test('should return EXACT on find the exact ip match', () => {
            const list = `127.0.0.2
2001:db8:11a3:9d7:0:0:0:0
192.168.0.1/8
127.0.0.1
127.0.0.3`;
            expect(getIpMatchListStatus('127.0.0.1', list))
                .toEqual(IP_MATCH_LIST_STATUS.EXACT);
        });

        test('should return CIDR on find the cidr match', () => {
            const list = `127.0.0.2
2001:db8:11a3:9d7:0:0:0:0
192.168.0.1/8
127.0.0.0/24
127.0.0.3`;
            expect(getIpMatchListStatus('127.0.0.1', list))
                .toEqual(IP_MATCH_LIST_STATUS.CIDR);
        });

        test('should return NOT_FOUND if the ip is not in the list', () => {
            const list = `127.0.0.1
2001:db8:11a3:9d7:0:0:0:0
192.168.0.1/8
127.0.0.2
127.0.0.3`;
            expect(getIpMatchListStatus('127.0.0.4', list))
                .toEqual(IP_MATCH_LIST_STATUS.NOT_FOUND);
        });

        test('should return the first EXACT or CIDR match in the list', () => {
            const list1 = `2001:db8:11a3:9d7:0:0:0:0
127.0.0.1
127.0.0.8/24
127.0.0.3`;
            expect(getIpMatchListStatus('127.0.0.1', list1))
                .toEqual(IP_MATCH_LIST_STATUS.EXACT);

            const list2 = `2001:db8:11a3:9d7:ffff:ffff:ffff:ffff
2001:0db8:11a3:09d7:0000:0000:0000:0000/64
127.0.0.0/24
127.0.0.1
127.0.0.8/24
127.0.0.3`;
            expect(getIpMatchListStatus('127.0.0.1', list2))
                .toEqual(IP_MATCH_LIST_STATUS.CIDR);
        });
    });

    describe('IPv6', () => {
        test('should return EXACT on find the exact ip match', () => {
            const list = `127.0.0.0
2001:db8:11a3:9d7:0:0:0:0
2001:db8:11a3:9d7:ffff:ffff:ffff:ffff
127.0.0.1`;
            expect(getIpMatchListStatus('2001:db8:11a3:9d7:0:0:0:0', list))
                .toEqual(IP_MATCH_LIST_STATUS.EXACT);
        });

        test('should return EXACT on find the exact ip match of short and long notation', () => {
            const list = `127.0.0.0
192.168.0.1/8
2001:db8::
127.0.0.2`;
            expect(getIpMatchListStatus('2001:db8:0:0:0:0:0:0', list))
                .toEqual(IP_MATCH_LIST_STATUS.EXACT);
        });

        test('should return CIDR on find the cidr match', () => {
            const list1 = `2001:0db8:11a3:09d7:0000:0000:0000:0000/64
127.0.0.1
127.0.0.2`;
            expect(getIpMatchListStatus('2001:db8:11a3:9d7:0:0:0:0', list1))
                .toEqual(IP_MATCH_LIST_STATUS.CIDR);

            const list2 = `2001:0db8::/16
127.0.0.0
2001:db8:11a3:9d7:0:0:0:0
2001:db8::
2001:db8:11a3:9d7:ffff:ffff:ffff:ffff
127.0.0.1`;
            expect(getIpMatchListStatus('2001:db1::', list2))
                .toEqual(IP_MATCH_LIST_STATUS.CIDR);
        });

        test('should return NOT_FOUND if the ip is not in the list', () => {
            const list = `2001:db8:11a3:9d7:0:0:0:0
2001:0db8:11a3:09d7:0000:0000:0000:0000/64
127.0.0.1
127.0.0.2`;
            expect(getIpMatchListStatus('::', list))
                .toEqual(IP_MATCH_LIST_STATUS.NOT_FOUND);
        });

        test('should return the first EXACT or CIDR match in the list', () => {
            const list1 = `2001:db8:11a3:9d7:0:0:0:0
2001:0db8:11a3:09d7:0000:0000:0000:0000/64
127.0.0.3`;
            expect(getIpMatchListStatus('2001:db8:11a3:9d7:0:0:0:0', list1))
                .toEqual(IP_MATCH_LIST_STATUS.EXACT);

            const list2 = `2001:0db8:11a3:09d7:0000:0000:0000:0000/64
2001:db8:11a3:9d7:0:0:0:0
127.0.0.3`;
            expect(getIpMatchListStatus('2001:db8:11a3:9d7:0:0:0:0', list2))
                .toEqual(IP_MATCH_LIST_STATUS.CIDR);
        });
    });

    describe('Empty list or IP', () => {
        test('should return NOT_FOUND on empty ip', () => {
            const list = `127.0.0.0
2001:db8:11a3:9d7:0:0:0:0
2001:db8:11a3:9d7:ffff:ffff:ffff:ffff
127.0.0.1`;
            expect(getIpMatchListStatus('', list))
                .toEqual(IP_MATCH_LIST_STATUS.NOT_FOUND);
        });

        test('should return NOT_FOUND on empty list', () => {
            const list = '';
            expect(getIpMatchListStatus('127.0.0.1', list))
                .toEqual(IP_MATCH_LIST_STATUS.NOT_FOUND);
        });
    });
});

describe('sortIp', () => {
    describe('ipv4', () => {
        test('one octet differ', () => {
            const arr = [
                '127.0.2.0',
                '127.0.3.0',
                '127.0.1.0',
            ];
            const sortedArr = [
                '127.0.1.0',
                '127.0.2.0',
                '127.0.3.0',
            ];
            expect(arr.sort(sortIp)).toStrictEqual(sortedArr);
        });
        test('few octets differ', () => {
            const arr = [
                '192.168.11.10',
                '192.168.10.0',
                '192.168.11.11',
                '192.168.10.10',
                '192.168.1.10',
                '192.168.0.1',
                '192.168.1.0',
                '192.168.1.1',
                '192.168.11.0',
                '192.168.0.10',
                '192.168.10.11',
                '192.168.0.11',
                '192.168.1.11',
                '192.168.0.0',
                '192.168.10.1',
                '192.168.11.1',
            ];
            const sortedArr = [
                '192.168.0.0',
                '192.168.0.1',
                '192.168.0.10',
                '192.168.0.11',
                '192.168.1.0',
                '192.168.1.1',
                '192.168.1.10',
                '192.168.1.11',
                '192.168.10.0',
                '192.168.10.1',
                '192.168.10.10',
                '192.168.10.11',
                '192.168.11.0',
                '192.168.11.1',
                '192.168.11.10',
                '192.168.11.11',
            ];
            expect(arr.sort(sortIp)).toStrictEqual(sortedArr);

            // Example from issue https://github.com/AdguardTeam/AdGuardHome/issues/1778#issuecomment-640937599
            const arr2 = [
                '192.168.2.11',
                '192.168.3.1',
                '192.168.2.100',
                '192.168.2.2',
                '192.168.2.1',
                '192.168.2.10',
                '192.168.2.99',
                '192.168.2.200',
                '192.168.2.199',
            ];
            const sortedArr2 = [
                '192.168.2.1',
                '192.168.2.2',
                '192.168.2.10',
                '192.168.2.11',
                '192.168.2.99',
                '192.168.2.100',
                '192.168.2.199',
                '192.168.2.200',
                '192.168.3.1',
            ];
            expect(arr2.sort(sortIp)).toStrictEqual(sortedArr2);
        });
    });
    describe('ipv6', () => {
        test('only long form', () => {
            const arr = [
                '2001:db8:11a3:9d7:0:0:0:2',
                '2001:db8:11a3:9d7:0:0:0:3',
                '2001:db8:11a3:9d7:0:0:0:1',
            ];
            const sortedArr = [
                '2001:db8:11a3:9d7:0:0:0:1',
                '2001:db8:11a3:9d7:0:0:0:2',
                '2001:db8:11a3:9d7:0:0:0:3',
            ];
            expect(arr.sort(sortIp)).toStrictEqual(sortedArr);
        });
        test('only short form', () => {
            const arr = [
                '2001:db8::',
                '2001:db7::',
                '2001:db9::',
            ];
            const sortedArr = [
                '2001:db7::',
                '2001:db8::',
                '2001:db9::',
            ];
            expect(arr.sort(sortIp)).toStrictEqual(sortedArr);
        });
        test('long and short forms', () => {
            const arr = [
                '2001:db8::',
                '2001:db7:11a3:9d7:0:0:0:2',
                '2001:db6:11a3:9d7:0:0:0:1',
                '2001:db6::',
                '2001:db7:11a3:9d7:0:0:0:1',
                '2001:db7::',
            ];
            const sortedArr = [
                '2001:db6::',
                '2001:db6:11a3:9d7:0:0:0:1',
                '2001:db7::',
                '2001:db7:11a3:9d7:0:0:0:1',
                '2001:db7:11a3:9d7:0:0:0:2',
                '2001:db8::',
            ];
            expect(arr.sort(sortIp)).toStrictEqual(sortedArr);
        });
    });
    describe('ipv4 and ipv6', () => {
        test('ipv6 long form', () => {
            const arr = [
                '127.0.0.3',
                '2001:db8:11a3:9d7:0:0:0:1',
                '2001:db8:11a3:9d7:0:0:0:3',
                '127.0.0.1',
                '2001:db8:11a3:9d7:0:0:0:2',
                '127.0.0.2',
            ];
            const sortedArr = [
                '127.0.0.1',
                '127.0.0.2',
                '127.0.0.3',
                '2001:db8:11a3:9d7:0:0:0:1',
                '2001:db8:11a3:9d7:0:0:0:2',
                '2001:db8:11a3:9d7:0:0:0:3',
            ];
            expect(arr.sort(sortIp)).toStrictEqual(sortedArr);
        });
        test('ipv6 short form', () => {
            const arr = [
                '2001:db8:11a3:9d7::1',
                '127.0.0.3',
                '2001:db8:11a3:9d7::3',
                '127.0.0.1',
                '2001:db8:11a3:9d7::2',
                '127.0.0.2',
            ];
            const sortedArr = [
                '127.0.0.1',
                '127.0.0.2',
                '127.0.0.3',
                '2001:db8:11a3:9d7::1',
                '2001:db8:11a3:9d7::2',
                '2001:db8:11a3:9d7::3',
            ];
            expect(arr.sort(sortIp)).toStrictEqual(sortedArr);
        });
        test('ipv6 long and short forms', () => {
            const arr = [
                '2001:db8:11a3:9d7::1',
                '127.0.0.3',
                '2001:db8:11a3:9d7:0:0:0:2',
                '127.0.0.1',
                '2001:db8:11a3:9d7::3',
                '127.0.0.2',
            ];
            const sortedArr = [
                '127.0.0.1',
                '127.0.0.2',
                '127.0.0.3',
                '2001:db8:11a3:9d7::1',
                '2001:db8:11a3:9d7:0:0:0:2',
                '2001:db8:11a3:9d7::3',
            ];
            expect(arr.sort(sortIp)).toStrictEqual(sortedArr);
        });
        test('always put ipv4 before ipv6', () => {
            const arr = [
                '::1',
                '0.0.0.2',
                '127.0.0.1',
                '::2',
                '2001:db8:11a3:9d7:0:0:0:2',
                '0.0.0.1',
                '2001:db8:11a3:9d7::1',
            ];
            const sortedArr = [
                '0.0.0.1',
                '0.0.0.2',
                '127.0.0.1',
                '::1',
                '::2',
                '2001:db8:11a3:9d7::1',
                '2001:db8:11a3:9d7:0:0:0:2',
            ];
            expect(arr.sort(sortIp)).toStrictEqual(sortedArr);
        });
    });
    describe('cidr', () => {
        test('only ipv4 cidr', () => {
            const arr = [
                '192.168.0.1/9',
                '192.168.0.1/7',
                '192.168.0.1/8',
            ];
            const sortedArr = [
                '192.168.0.1/7',
                '192.168.0.1/8',
                '192.168.0.1/9',
            ];
            expect(arr.sort(sortIp)).toStrictEqual(sortedArr);
        });
        test('ipv4 and cidr ipv4', () => {
            const arr = [
                '192.168.0.1/9',
                '192.168.0.1',
                '192.168.0.1/32',
                '192.168.0.1/7',
                '192.168.0.1/8',
            ];
            const sortedArr = [
                '192.168.0.1/7',
                '192.168.0.1/8',
                '192.168.0.1/9',
                '192.168.0.1/32',
                '192.168.0.1',
            ];
            expect(arr.sort(sortIp)).toStrictEqual(sortedArr);
        });
        test('only ipv6 cidr', () => {
            const arr = [
                '2001:db8:11a3:9d7::1/32',
                '2001:db8:11a3:9d7::1/64',
                '2001:db8:11a3:9d7::1/128',
                '2001:db8:11a3:9d7::1/24',
            ];
            const sortedArr = [
                '2001:db8:11a3:9d7::1/24',
                '2001:db8:11a3:9d7::1/32',
                '2001:db8:11a3:9d7::1/64',
                '2001:db8:11a3:9d7::1/128',
            ];
            expect(arr.sort(sortIp)).toStrictEqual(sortedArr);
        });
        test('ipv6 and cidr ipv6', () => {
            const arr = [
                '2001:db8:11a3:9d7::1/32',
                '2001:db8:11a3:9d7::1',
                '2001:db8:11a3:9d7::1/64',
                '2001:db8:11a3:9d7::1/128',
                '2001:db8:11a3:9d7::1/24',
            ];
            const sortedArr = [
                '2001:db8:11a3:9d7::1/24',
                '2001:db8:11a3:9d7::1/32',
                '2001:db8:11a3:9d7::1/64',
                '2001:db8:11a3:9d7::1/128',
                '2001:db8:11a3:9d7::1',
            ];
            expect(arr.sort(sortIp)).toStrictEqual(sortedArr);
        });
    });
    describe('invalid input', () => {
        const originalError = console.error;

        beforeEach(() => {
            console.error = jest.fn();
        });

        afterEach(() => {
            expect(console.error).toHaveBeenCalled();
            console.error = originalError;
        });

        test('invalid strings', () => {
            const arr = ['invalid ip', 'invalid cidr'];
            expect(arr.sort(sortIp)).toStrictEqual(arr);
        });
        test('invalid ip', () => {
            const arr = ['127.0.0.2.', '.127.0.0.1.', '.2001:db8:11a3:9d7:0:0:0:0'];
            expect(arr.sort(sortIp)).toStrictEqual(arr);
        });
        test('invalid cidr', () => {
            const arr = ['127.0.0.2/33', '2001:db8:11a3:9d7:0:0:0:0/129'];
            expect(arr.sort(sortIp)).toStrictEqual(arr);
        });
        test('valid and invalid ip', () => {
            const arr = ['127.0.0.4.', '127.0.0.1', '.127.0.0.3', '127.0.0.2'];
            expect(arr.sort(sortIp)).toStrictEqual(arr);
        });
    });
    describe('mixed', () => {
        test('ipv4, ipv6 in short and long forms and cidr', () => {
            const arr = [
                '2001:db8:11a3:9d7:0:0:0:1/32',
                '192.168.1.2',
                '127.0.0.2',
                '2001:db8:11a3:9d7::1/128',
                '2001:db8:11a3:9d7:0:0:0:1',
                '127.0.0.1/12',
                '192.168.1.1',
                '2001:db8::/32',
                '2001:db8:11a3:9d7::1/24',
                '192.168.1.2/12',
                '2001:db7::/32',
                '127.0.0.1',
                '2001:db8:11a3:9d7:0:0:0:2',
                '192.168.1.1/24',
                '2001:db7::/64',
                '2001:db7::',
                '2001:db8::',
                '2001:db8:11a3:9d7:0:0:0:1/128',
                '192.168.1.1/12',
                '127.0.0.1/32',
                '::1',
            ];
            const sortedArr = [
                '127.0.0.1/12',
                '127.0.0.1/32',
                '127.0.0.1',
                '127.0.0.2',
                '192.168.1.1/12',
                '192.168.1.1/24',
                '192.168.1.1',
                '192.168.1.2/12',
                '192.168.1.2',
                '::1',
                '2001:db7::/32',
                '2001:db7::/64',
                '2001:db7::',
                '2001:db8::/32',
                '2001:db8::',
                '2001:db8:11a3:9d7::1/24',
                '2001:db8:11a3:9d7:0:0:0:1/32',
                '2001:db8:11a3:9d7::1/128',
                '2001:db8:11a3:9d7:0:0:0:1/128',
                '2001:db8:11a3:9d7:0:0:0:1',
                '2001:db8:11a3:9d7:0:0:0:2',
            ];
            expect(arr.sort(sortIp)).toStrictEqual(sortedArr);
        });
    });
});
