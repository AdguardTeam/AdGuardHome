import { getIpMatchListStatus } from '../helpers/helpers';
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
