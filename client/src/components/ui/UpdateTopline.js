import React, { Fragment } from 'react';
import PropTypes from 'prop-types';
import { Trans, withTranslation } from 'react-i18next';

import Topline from './Topline';

const UpdateTopline = (props) => (
    <Topline type="info">
        <Fragment>
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
            {props.canAutoUpdate
                && <button
                    type="button"
                    className="btn btn-sm btn-primary ml-3"
                    onClick={props.getUpdate}
                    disabled={props.processingUpdate}
                >
                    <Trans>update_now</Trans>
                </button>
            }
        </Fragment>
    </Topline>
);

UpdateTopline.propTypes = {
    version: PropTypes.string,
    url: PropTypes.string.isRequired,
    canAutoUpdate: PropTypes.bool,
    getUpdate: PropTypes.func,
    processingUpdate: PropTypes.bool,
};

export default withTranslation()(UpdateTopline);
