import React from 'react';
import { withTranslation, Trans } from 'react-i18next';

import Card from './Card';

interface StatusProps {
    message: string;
    buttonMessage?: string;
    reloadPage?: (...args: unknown[]) => unknown;
}

const Status = ({ message, buttonMessage, reloadPage }: StatusProps) => (
    <div className="status">
        <Card bodyType="card-body card-body--status">
            <div className="h4 font-weight-light mb-4">
                <Trans>{message}</Trans>
            </div>
            {buttonMessage && (
                <button className="btn btn-success" onClick={reloadPage}>
                    <Trans>{buttonMessage}</Trans>
                </button>
            )}
        </Card>
    </div>
);

export default withTranslation()(Status);
