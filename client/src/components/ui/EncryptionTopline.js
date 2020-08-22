import React from 'react';
import { Trans } from 'react-i18next';
import isAfter from 'date-fns/is_after';
import addDays from 'date-fns/add_days';
import { useSelector } from 'react-redux';
import Topline from './Topline';
import { EMPTY_DATE } from '../../helpers/constants';

const EncryptionTopline = () => {
    const not_after = useSelector((state) => state.encryption.not_after);

    if (not_after === EMPTY_DATE) {
        return null;
    }

    const isAboutExpire = isAfter(addDays(Date.now(), 30), not_after);
    const isExpired = isAfter(Date.now(), not_after);

    if (isExpired) {
        return (
            <Topline type="danger">
                <Trans components={[<a href="#encryption" key="0">link</a>]}>
                    topline_expired_certificate
                </Trans>
            </Topline>
        );
    }

    if (isAboutExpire) {
        return (
            <Topline type="warning">
                <Trans components={[<a href="#encryption" key="0">link</a>]}>
                    topline_expiring_certificate
                </Trans>
            </Topline>
        );
    }

    return false;
};

export default EncryptionTopline;
