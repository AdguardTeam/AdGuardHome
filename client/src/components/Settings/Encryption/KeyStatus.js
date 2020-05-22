import React, { Fragment } from 'react';
import PropTypes from 'prop-types';
import { withTranslation, Trans } from 'react-i18next';

const KeyStatus = ({ validKey, keyType }) => (
    <Fragment>
        <div className="form__label form__label--bold">
            <Trans>encryption_status</Trans>:
        </div>
        <ul className="encryption__list">
            <li className={validKey ? 'text-success' : 'text-danger'}>
                {validKey ? (
                    <Trans values={{ type: keyType }}>
                        encryption_key_valid
                    </Trans>
                ) : (
                    <Trans values={{ type: keyType }}>
                        encryption_key_invalid
                    </Trans>
                )}
            </li>
        </ul>
    </Fragment>
);

KeyStatus.propTypes = {
    validKey: PropTypes.bool.isRequired,
    keyType: PropTypes.string.isRequired,
};

export default withTranslation()(KeyStatus);
