import React from 'react';
import Link from '@docusaurus/Link';
import LayoutProviders from '@theme/LayoutProviders';
import Footer from '@theme/Footer';
import index from './index.module.css';
import GithubIcon from '../assets/icons/github.svg';
import DiscordIcon from '../assets/icons/discord.svg';
import BrowserOnly from "@docusaurus/core/lib/client/exports/BrowserOnly";

function Chip({ text }) {
    return <div className={index.chip}>
        {text}
    </div>
}

function Nav() {
    return  <ul className={index.nav}>
        <li className={index.linkItem}>
            <Link to="/CLI/atlas">
                Docs
            </Link>
        </li>
        <li className={index.linkItem}>
            <Link to={"/cli/getting-started/setting-up"}>
                Getting Started
            </Link>
        </li>
    </ul>
}

function SocialLinks() {
    return <div className={index.socialLinks}>
        <a href="https://github.com/ariga/atlas" target="_blank">
            <GithubIcon />
        </a>

        <a href="https://discord.com/invite/QhsmBAWzrC" target="_blank">
            <DiscordIcon />
        </a>
    </div>
}

function Header() {
    return <div className={index.header}>
        <Nav />
        <SocialLinks />
    </div>
}

function AtlasButton({ link, text, type, style }) {
    return (
        <button style={{...style}} className={index[type]}>
            <a className={index[`${type}Text`]} href={link}>{text}</a>
        </button>
    )
}

export default function () {
    return <LayoutProviders>
        {/* first slide */}
        <div id="slide1" className={index.slide1} style={{backgroundImage:'url(https://atlasgo.io/uploads/landing/background.png)'}}>
            <Header />
            <div className={index.rowContainer}>
                <div className={index.slide1LeftSide}>
                    <h2 className={index.title}
                        style={{ color: "#82C7FF" }}>Manage your <br /> database schemas with <span style={{ color: "white" }}>Atlas CLI</span></h2>

                    <p className={index.paragraph} style={{ color: "#DFF1FF", textAlign: "left", width: "90%" }}>
                        Atlas CLI is an open source tool that helps developers
                        manage their database schemas by applying modern
                        DevOps principles. Contrary to existing tools, Atlas
                        intelligently plans schema migrations for you, based
                        on your desired state.
                    </p>
                    <AtlasButton style={{"marginTop": "5%"}} text="Get Started"  link="/cli/getting-started/setting-up" type="primaryButton"/>
                </div>
                <div className={index.imageContainer}>
                    <img src="https://atlasgo.io/uploads/images/help_cmd.png" alt=""/>
                </div>
            </div>
        </div>

        {/* 2nd slide */}
        <div className={index.slide2}>
            <div className={index.container}>
                <section className={index.sectionNoMargin}>
                    <h2 className={index.titleSecondary} style={{ textAlign: "center", color: "#000939" }}>
                        Define your schemas using the <span style={{color: "#2064E9"}}>Atlas DDL</span>
                    </h2>
                    <p className={index.paragraphSecondary}
                       style={{ color: "#757889" }}>
                        Atlas DDL is a declarative, Terraform-like configuration language designed to capture an
                        organization’s data topology. Currently, it supports defining schemas for SQL databases such as
                        MySQL, Postgres, SQLite and MariaDB.
                    </p>
                </section>

                <button className={index.textButton}>
                    <Link to="/ddl/intro">Read More <span>&#8594;</span></Link>
                </button>

                <BrowserOnly>
                    {() => {
                        const mobile = window.innerWidth < 768;
                        const erdImage = mobile ? "https://atlasgo.io/uploads/images/erd_2.png" : "https://atlasgo.io/uploads/erd-180122.png";
                        return <img style={{margin: "20px 0" }} src={erdImage} alt="erd"/>
                    }}
                </BrowserOnly>

                <section className={index.section} style={{ marginTop: "10px" }}>
                   <h2 className={index.titleSecondary} style={{ textAlign: "center", color: "#000939" }}>
                       Visualize your schemas <br/> using the <span style={{color: "#2064E9"}}>Atlas UI</span>
                   </h2>
                   <p className={index.paragraphSecondary}
                      style={{ color: "#757889" }}>
                       With a single command-line flag, Atlas opens up as a rich user interface to visualize and manage your data topology.
                   </p>
               </section>
                <AtlasButton text="Get Started" link="/ui/intro" type="secondaryButton"/>

                <img  className={index.dashboardImage} src="https://atlasgo.io/uploads/atlas-ui-180122.png" alt="ent"/>
            </div>
        </div>

        {/* 4th slide   */}
        <div className={index.slide4}>
            <div className={index.container}>
                <section className={index.section}>
                    <h1 style={{color: "#000939"}} className={index.titleSecondary}>Migrate,&nbsp;<span style={{color: "#2064E9"}}>your way.</span></h1>
                    <p className={index.paragraphSecondary}>Atlas provides the user with two types of migrations - declarative and versioned.</p>
               </section>

                <section className={index.section}>
                    <h2 style={{color: "#000939"}} className={index.subtitle}>Declarative Migrations</h2>
                    <p className={index.paragraphSecondary}>Declarative migrations are migrations in which the user provides the desired state, and Atlas gets your schema there instantly.</p>
                </section>

                <section className={index.section}>
                    <div className={index.subtitleWithChipWrapper}>
                        <h2 style={{color: "#000939", marginRight: "10px"}} className={index.subtitleMargin}>Versioned Migrations</h2>
                        <Chip text="Coming soon" />
                    </div>
                    <p className={index.paragraphSecondary}>Atlas offers you an alternative workflow, in which migrations are explicitly defined and
                        assigned a version. Atlas can then bring a schema to the desired version by following
                        the migrations between the current version and the specified one.</p>
                </section>
            </div>
        </div>
        <Footer />
    </LayoutProviders>
}
