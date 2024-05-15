// FROM IONIC CODEBASE: https://github.com/ionic-team/ionic-docs/blob/main/src/components/global/DocsCard/index.tsx

import React from 'react';
import clsx from 'clsx';
import Link from '@docusaurus/Link';
import useBaseUrl from '@docusaurus/useBaseUrl';

import styles from './styles.modules.scss';

interface Props extends React.HTMLAttributes<HTMLDivElement> {
    href?: string;
    header?: string;
    icon?: string;
    hoverIcon?: string;
    iconset?: string;
    img?: string;
    size?: 'md' | 'lg';
    iconPosition?: 'left' | 'right'; // Add iconPosition prop
}

function DocsCard(props: Props): JSX.Element {
    const isStatic = typeof props.href === 'undefined';
    const isOutbound = typeof props.href !== 'undefined' ? /^http/.test(props.href) : false;
    const header = props.header === 'undefined' ? null : <header className="Card-header">{props.header}</header>;
    const hoverIcon = props.hoverIcon || props.icon;
    const iconPositionClass = props.iconPosition === 'left' ? 'icon-left' : props.iconPosition === 'right' ? 'icon-right' : 'icon-center';
    const hasIconCenter = props.icon && props.iconPosition === 'center';
    const hasIcon = props.icon && props.iconPosition === 'center';

    const content = (
        <>
            {props.img && <img src={useBaseUrl(props.img)} className="Card-image" />}
            <div  className={clsx("Card-container", { "has-icon-center": hasIconCenter })}>
                {props.icon && props.iconPosition === 'left' && (
                    <div className="Card-icon-row">
                        <div className={`Card-icon ${iconPositionClass}`}>{props.icon}</div>
                        {props.header && header}
                    </div>
                )}
                {props.icon && props.iconPosition === 'right' && (
                    <div className="Card-icon-row">
                        {props.header && header}
                        <div className={`Card-icon ${iconPositionClass}`}>{props.icon}</div>
                    </div>
                )}
                {props.icon && props.iconPosition === 'center' && (
                    <div className="Card-icon-row">
                        <div className ="icon-center">
                        <div className={`Card-icon ${iconPositionClass}`}>{props.icon}</div>
                        <div className={`${iconPositionClass} Card-header`}>{props.header && header}
                        </div>
                    </div>
                    </div>
                )}
                {!props.icon && props.iconPosition !== 'center' && props.header && header}
                <div className="Card-content">{props.children}</div>
            </div>
        </>
    );

    const className = clsx({
        'Card-with-image': typeof props.img !== 'undefined',
        'Card-without-image': typeof props.img === 'undefined',
        'Card-size-lg': props.size === 'lg',
        [props.className]: props.className,
    });

    if (isStatic) {
        return (
            <docs-card class={className}>
                <div className={clsx(styles.card, 'docs-card')}>{content}</div>
            </docs-card>
        );
    }

    if (isOutbound) {
        return (
            <docs-card class={className}>
                <a className={clsx(styles.card, 'docs-card')} href={props.href} target="_blank">
                    {content}
                </a>
            </docs-card>
        );
    }

    return (
        <docs-card class={className}>
            <Link to={props.href} className={clsx(styles.card, 'docs-card')}>
                {content}
            </Link>
        </docs-card>
    );
}

export default DocsCard;
