import React, { Fragment } from 'react';
import { withTranslation, Trans } from 'react-i18next';

interface KeyStatusProps {
    validKey: boolean;
    keyType: string;
}

const KeyStatus = ({ validKey, keyType }: KeyStatusProps) => (
    <Fragment>
        <div className="form__label form__label--bold">
            <Trans>encryption_status</Trans>:
        </div>

        <ul className="encryption__list">
            <li className={validKey ? 'text-success' : 'text-danger'}>
                {validKey ? (
                    <Trans values={{ type: keyType }}>encryption_key_valid</Trans>
                ) : (
                    <Trans values={{ type: keyType }}>encryption_key_invalid</Trans>
                )}
            </li>
        </ul>
    </Fragment>
);

export default withTranslation()(KeyStatus);
