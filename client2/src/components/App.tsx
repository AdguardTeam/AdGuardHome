import { observer } from 'mobx-react-lite';
import React, { FC, useContext } from 'react';
import Store from 'Store';
import Icons from 'Lib/theme/Icons';

const App: FC = observer(() => {
    const store = useContext(Store);
    return (
        <div>
            {store.ui.currentLang}
            <Icons/>
        </div>
    );
});

export default App;
