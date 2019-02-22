import React from 'react';
import PropTypes from 'prop-types';
import { Trans, withNamespaces } from 'react-i18next';

import Topline from './Topline';

const UpdateTopline = props => (
    <Topline type="info">
        <Trans
            values={{ version: props.version }}
            components={[
                <a href={props.url} target="_blank" rel="noopener noreferrer" key="0">
                    Click here
                </a>,
            ]}
        >
            update_announcement
        </Trans>
    </Topline>
);

UpdateTopline.propTypes = {
    version: PropTypes.string.isRequired,
    url: PropTypes.string.isRequired,
};

export default withNamespaces()(UpdateTopline);
