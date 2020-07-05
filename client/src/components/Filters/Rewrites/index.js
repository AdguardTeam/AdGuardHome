import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import { Trans, withTranslation } from 'react-i18next';

import Table from './Table';
import Modal from './Modal';
import Card from '../../ui/Card';
import PageTitle from '../../ui/PageTitle';

class Rewrites extends Component {
    componentDidMount() {
        this.props.getRewritesList();
    }

    handleSubmit = (values) => {
        this.props.addRewrite(values);
    };

    handleDelete = (values) => {
        // eslint-disable-next-line no-alert
        if (window.confirm(this.props.t('rewrite_confirm_delete', { key: values.domain }))) {
            this.props.deleteRewrite(values);
        }
    };

    render() {
        const {
            t,
            rewrites,
            toggleRewritesModal,
        } = this.props;

        const {
            list,
            isModalOpen,
            processing,
            processingAdd,
            processingDelete,
        } = rewrites;

        return (
            <Fragment>
                <PageTitle
                    title={t('dns_rewrites')}
                    subtitle={t('rewrite_desc')}
                />
                <Card
                    id="rewrites"
                    bodyType="card-body box-body--settings"
                >
                    <Fragment>
                        <Table
                            list={list}
                            processing={processing}
                            processingAdd={processingAdd}
                            processingDelete={processingDelete}
                            handleDelete={this.handleDelete}
                        />

                        <button
                            type="button"
                            className="btn btn-success btn-standard mt-3"
                            onClick={() => toggleRewritesModal()}
                            disabled={processingAdd}
                        >
                            <Trans>rewrite_add</Trans>
                        </button>

                        <Modal
                            isModalOpen={isModalOpen}
                            toggleRewritesModal={toggleRewritesModal}
                            handleSubmit={this.handleSubmit}
                            processingAdd={processingAdd}
                            processingDelete={processingDelete}
                        />
                    </Fragment>
                </Card>
            </Fragment>
        );
    }
}

Rewrites.propTypes = {
    t: PropTypes.func.isRequired,
    getRewritesList: PropTypes.func.isRequired,
    toggleRewritesModal: PropTypes.func.isRequired,
    addRewrite: PropTypes.func.isRequired,
    deleteRewrite: PropTypes.func.isRequired,
    rewrites: PropTypes.object.isRequired,
};

export default withTranslation()(Rewrites);
