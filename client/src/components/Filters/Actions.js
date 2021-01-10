import React from 'react';
import PropTypes from 'prop-types';
import { withTranslation, Trans } from 'react-i18next';

const Actions = ({
    handleAdd, handleRefresh, processingRefreshFilters, whitelist,
}) => <div className="card-actions">
    <button
            className="btn btn-success btn-standard mr-2 btn-large mb-2"
            type="submit"
            onClick={handleAdd}
    >
        {whitelist ? <Trans>add_allowlist</Trans> : <Trans>add_blocklist</Trans>}
    </button>
    <button
            className="btn btn-primary btn-standard mb-2"
            type="submit"
            onClick={handleRefresh}
            disabled={processingRefreshFilters}
    >
        <Trans>check_updates_btn</Trans>
    </button>
</div>;

Actions.propTypes = {
    handleAdd: PropTypes.func.isRequired,
    handleRefresh: PropTypes.func.isRequired,
    processingRefreshFilters: PropTypes.bool.isRequired,
    whitelist: PropTypes.bool,
};

export default withTranslation()(Actions);
