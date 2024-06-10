import React from 'react';
import { Trans } from 'react-i18next';

import { HashLink as Link } from 'react-router-hash-link';

const AnonymizerNotification = () => (
    <div className="alert alert-primary mt-6">
        <Trans
            components={[
                <strong key="0">text</strong>,

                <Link to="/settings#logs-config" key="1">
                    link
                </Link>,
            ]}>
            anonymizer_notification
        </Trans>
    </div>
);

export default AnonymizerNotification;
