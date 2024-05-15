// FROM IONIC CODEBASE: https://github.com/ionic-team/ionic-docs/blob/main/src/components/global/DocsCards/index.tsx
import React from 'react';

import './cards.css';

function DocsCards(props): JSX.Element {
    return <docs-cards class={props.className}>{props.children}</docs-cards>;
}

export default DocsCards;
