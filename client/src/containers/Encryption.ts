import { connect } from 'react-redux';
import { getTlsStatus, setTlsConfig, validateTlsConfig } from '../actions/encryption';

import Encryption from '../components/Settings/Encryption';

const mapStateToProps = (state: any) => {
    const { encryption } = state;
    const props = {
        encryption,
    };
    return props;
};

const mapDispatchToProps = {
    getTlsStatus,
    setTlsConfig,
    validateTlsConfig,
};

export default connect(mapStateToProps, mapDispatchToProps)(Encryption);
