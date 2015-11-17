import React from 'react'
import classNames from 'classnames'

import Page from './Page'
import Header from './Header'
import Footer from './Footer'


export default function MainPage(props) {
  return (
    <Page className={classNames('page', props.className)} title={props.title} heimPage={props.heimPage}>
      <Header />
      {props.nav || null}
      <div className="container main">
        {props.children}
      </div>
      <Footer />
    </Page>
  )
}

MainPage.propTypes = {
  className: React.PropTypes.string,
  title: React.PropTypes.string,
  heimPage: React.PropTypes.string,
  nav: React.PropTypes.node,
  children: React.PropTypes.node,
}
