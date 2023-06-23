import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import { Trans, withTranslation } from 'react-i18next';

import Table from './Table';
import Modal from './Modal';
import Card from '../../ui/Card';
import PageTitle from '../../ui/PageTitle';
import { MODAL_TYPE } from '../../../helpers/constants';

class Rewrites extends Component {
    componentDidMount() {
        this.props.getRewritesList();
    }

    handleDelete = (values) => {
        // eslint-disable-next-line no-alert
        if (window.confirm(this.props.t('rewrite_confirm_delete', { key: values.domain }))) {
            this.props.deleteRewrite(values);
        }
    };

    handleSubmit = (values) => {
        const { modalType, currentRewrite } = this.props.rewrites;

        if (modalType === MODAL_TYPE.EDIT_REWRITE && currentRewrite) {
            this.props.updateRewrite({
                target: currentRewrite,
                update: values,
            });
        } else {
            this.props.addRewrite(values);
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
            processingUpdate,
            modalType,
            currentRewrite,
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
                            processingUpdate={processingUpdate}
                            handleDelete={this.handleDelete}
                            toggleRewritesModal={toggleRewritesModal}
                        />

                        <button
                            type="button"
                            className="btn btn-success btn-standard mt-3"
                            onClick={() => toggleRewritesModal({ type: MODAL_TYPE.ADD_REWRITE })}
                            disabled={processingAdd}
                        >
                            <Trans>rewrite_add</Trans>
                        </button>

                        <Modal
                            isModalOpen={isModalOpen}
                            modalType={modalType}
                            toggleRewritesModal={toggleRewritesModal}
                            handleSubmit={this.handleSubmit}
                            processingAdd={processingAdd}
                            processingDelete={processingDelete}
                            processingUpdate={processingUpdate}
                            currentRewrite={currentRewrite}
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
    updateRewrite: PropTypes.func.isRequired,
    rewrites: PropTypes.object.isRequired,
};

export default withTranslation()(Rewrites);
