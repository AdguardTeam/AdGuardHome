import React, { Fragment } from 'react';
import { Trans } from 'react-i18next';

import { HashLink as Link } from 'react-router-hash-link';

import Card from '../ui/Card';

const Disabled = () => (
    <Fragment>
        <div className="page-header">
            <h1 className="page-title page-title--large">
                <Trans>query_log</Trans>
            </h1>
        </div>

        <Card>
            <div className="lead text-center py-6">
                <Trans
                    components={[
                        <Link to="/settings#logs-config" key="0">
                            link
                        </Link>,
                    ]}>
                    query_log_disabled
                </Trans>
            </div>
        </Card>
    </Fragment>
);

export default Disabled;
