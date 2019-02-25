import React from 'react';
import PropTypes from 'prop-types';
import { Trans, withNamespaces } from 'react-i18next';
import isAfter from 'date-fns/is_after';
import addDays from 'date-fns/add_days';

import Topline from './Topline';
import { EMPTY_DATE } from '../../helpers/constants';

const EncryptionTopline = (props) => {
    if (props.notAfter === EMPTY_DATE) {
        return false;
    }

    const isAboutExpire = isAfter(addDays(Date.now(), 30), props.notAfter);
    const isExpired = isAfter(Date.now(), props.notAfter);

    if (isExpired) {
        return (
            <Topline type="danger">
                <Trans components={[<a href="#settings" key="0">link</a>]}>
                    topline_expired_certificate
                </Trans>
            </Topline>
        );
    } else if (isAboutExpire) {
        return (
            <Topline type="warning">
                <Trans components={[<a href="#settings" key="0">link</a>]}>
                    topline_expiring_certificate
                </Trans>
            </Topline>
        );
    }

    return false;
};

EncryptionTopline.propTypes = {
    notAfter: PropTypes.string.isRequired,
};

export default withNamespaces()(EncryptionTopline);
