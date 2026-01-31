import React from 'react';
import { Trans } from 'react-i18next';
import isAfter from 'date-fns/is_after';
import addHours from 'date-fns/add_hours';
import differenceInHours from 'date-fns/difference_in_hours';
import { useSelector } from 'react-redux';

import Topline from './Topline';
import { EMPTY_DATE } from '../../helpers/constants';
import { RootState } from '../../initialState';

const EXPIRATION_ENUM = {
    VALID: 'VALID',
    EXPIRED: 'EXPIRED',
    EXPIRING: 'EXPIRING',
};

const EXPIRATION_STATE = {
    [EXPIRATION_ENUM.EXPIRED]: {
        toplineType: 'danger',
        i18nKey: 'topline_expired_certificate',
    },
    [EXPIRATION_ENUM.EXPIRING]: {
        toplineType: 'warning',
        i18nKey: 'topline_expiring_certificate',
    },
};

const getExpirationFlags = (not_before: any, not_after: any) => {
    const REG_MIN_RATIO_VALIDITY_REMAINING = 0.333;
    const SHORT_MIN_RATIO_VALIDITY_REMAINING = 0.5;
    const SHORT_LIVED_HOURS = 10 * 24;

    const certLifetimeHours = differenceInHours(not_after, not_before);
    const expiringThreshold = certLifetimeHours < SHORT_LIVED_HOURS ?
        certLifetimeHours * (1 - SHORT_MIN_RATIO_VALIDITY_REMAINING) :
        certLifetimeHours * (1 - REG_MIN_RATIO_VALIDITY_REMAINING);
     
    const now = Date.now();

    const isExpiring = isAfter(now, addHours(not_before, expiringThreshold));
    const isExpired = isAfter(now, not_after);

    return {
        isExpiring,
        isExpired,
    };
};

const getExpirationEnumKey = (not_before: any, not_after: any) => {
    const { isExpiring, isExpired } = getExpirationFlags(not_before, not_after);

    if (isExpired) {
        return EXPIRATION_ENUM.EXPIRED;
    }

    if (isExpiring) {
        return EXPIRATION_ENUM.EXPIRING;
    }

    return EXPIRATION_ENUM.VALID;
};

const EncryptionTopline = () => {
    const not_before = useSelector((state: RootState) => state.encryption.not_before);
    const not_after = useSelector((state: RootState) => state.encryption.not_after);

    if (not_before === EMPTY_DATE || not_after === EMPTY_DATE) {
        return null;
    }

    const expirationStateKey = getExpirationEnumKey(not_before, not_after);

    if (expirationStateKey === EXPIRATION_ENUM.VALID) {
        return null;
    }

    const { toplineType, i18nKey } = EXPIRATION_STATE[expirationStateKey];

    return (
        <Topline type={toplineType}>
            <Trans
                components={[
                    <a href="#encryption" key="0">
                        link
                    </a>,
                ]}>
                {i18nKey}
            </Trans>
        </Topline>
    );
};

export default EncryptionTopline;
