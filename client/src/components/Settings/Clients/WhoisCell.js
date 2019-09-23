import React from 'react';
import PropTypes from 'prop-types';
import { Trans } from 'react-i18next';

const getFormattedWhois = (value) => {
    const keys = Object.keys(value);

    if (keys.length > 0) {
        return (
            keys.map(key => (
                <div key={key} title={value[key]}>
                    <Trans
                        values={{ value: value[key] }}
                        components={[<small key="0">text</small>]}
                    >
                        {key}
                    </Trans>
                </div>
            ))
        );
    }

    return '–';
};

const WhoisCell = ({ value }) => (
    <div className="logs__row logs__row--overflow">
        <span className="logs__text logs__text--wrap">
            {getFormattedWhois(value)}
        </span>
    </div>
);

WhoisCell.propTypes = {
    value: PropTypes.object.isRequired,
};

export default WhoisCell;
