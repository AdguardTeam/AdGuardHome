import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactModal from 'react-modal';
import { Trans, withNamespaces } from 'react-i18next';

import { MODAL_TYPE } from '../../helpers/constants';
import Form from './Form';
import '../ui/Modal.css';

ReactModal.setAppElement('#root');

class Modal extends Component {
    closeModal = () => {
        this.props.toggleModal();
    };

    render() {
        const {
            isOpen,
            processingAddFilter,
            processingConfigFilter,
            handleSubmit,
            modalType,
            currentFilterData,
        } = this.props;

        return (
            <ReactModal
                className="Modal__Bootstrap modal-dialog modal-dialog-centered"
                closeTimeoutMS={0}
                isOpen={isOpen}
                onRequestClose={this.closeModal}
            >
                <div className="modal-content">
                    <div className="modal-header">
                        <h4 className="modal-title">
                            {modalType === MODAL_TYPE.EDIT ? (
                                <Trans>edit_filter_title</Trans>
                            ) : (
                                <Trans>new_filter_btn</Trans>
                            )}
                        </h4>
                        <button type="button" className="close" onClick={this.closeModal}>
                            <span className="sr-only">Close</span>
                        </button>
                    </div>
                    <Form
                        initialValues={{ ...currentFilterData }}
                        onSubmit={handleSubmit}
                        processingAddFilter={processingAddFilter}
                        processingConfigFilter={processingConfigFilter}
                        closeModal={this.closeModal}
                    />
                </div>
            </ReactModal>
        );
    }
}

Modal.propTypes = {
    toggleModal: PropTypes.func.isRequired,
    isOpen: PropTypes.bool.isRequired,
    addFilter: PropTypes.func.isRequired,
    isFilterAdded: PropTypes.bool.isRequired,
    processingAddFilter: PropTypes.bool.isRequired,
    processingConfigFilter: PropTypes.bool.isRequired,
    handleSubmit: PropTypes.func.isRequired,
    modalType: PropTypes.string.isRequired,
    currentFilterData: PropTypes.object.isRequired,
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(Modal);
