/* eslint-disable react/no-multi-comp */

import React from 'react'
import Reflux from 'reflux'
import Immutable from 'immutable'
import moment from 'moment'


export default function initPlugins(roomName) {
  if (roomName === 'thedrawingroom' || roomName === 'lovenest' || roomName === 'has') {
    Heim.hook('page-bottom', () => {
      return (
        <style key="drawingroom-style" dangerouslySetInnerHTML={{__html: `
          .chat-pane.timestamps-visible {
            background: #333;
          }

          .main-pane .room .name,
          .info-pane .thread-list .thread .info .title {
            color: #222;
          }

          .chat-pane time {
            opacity: .5;
          }

          .main-pane .room .state,
          .nick {
            background: #e8e8e8 !important;
          }

          .message-emote {
            background: #f3f3f3 !important;
          }

          .mention-nick {
            color: #000 !important;
            font-weight: bold;
          }

          a {
            color: #444;
            text-decoration: none;
            font-weight: bold;
          }
        `}} />
      )
    })
  }

  if (roomName === 'space') {
    const Embed = require('./ui/Embed').default

    Heim.hook('main-sidebar', () => {
      return (
        <div key="norman" className="norman">
          <p>norman</p>
          <Embed kind="imgur" imgur_id="UKbitCO" />
        </div>
      )
    })

    Heim.hook('page-bottom', () => {
      return (
        <style key="norman-style" dangerouslySetInnerHTML={{__html: `
          .norman {
            opacity: .5;
            transition: opacity .15s ease;
          }

          .norman:hover {
            opacity: 1;
          }

          .norman p {
            margin: 0;
            font-size: 12px;
          }

          .norman .embed {
            width: 100% !important;
            height: 87px;
            border: none;
          }
        `}} />
      )
    })
  }

  if (roomName === 'music' || roomName === 'youtube' || roomName === 'rmusic' || roomName === 'listentothis') {
    const Embed = require('./ui/Embed').default
    const MessageText = require('./ui/MessageText').default

    let clientTimeOffset = 0
    Heim.chat.store.socket.on('receive', ev => {
      if (ev.type === 'ping-event') {
        clientTimeOffset = Date.now() / 1000 - ev.data.time
      }
    })

    const TVActions = Reflux.createActions([
      'changeVideo',
      'changeNotice',
    ])

    Heim.ui.createCustomPane('youtube-tv', {readOnly: true})

    const TVStore = Reflux.createStore({
      listenables: [
        TVActions,
        {chatChange: Heim.chat.store},
      ],

      init() {
        this.state = Immutable.fromJS({
          video: {
            time: 0,
            messageId: null,
            youtubeId: null,
            youtubeTime: 0,
            title: '',
          },
          notice: {
            time: 0,
            content: '',
          },
        })
      },

      getInitialState() {
        return this.state
      },

      changeVideo(video) {
        this.state = this.state.set('video', Immutable.fromJS(video))
        this.trigger(this.state)
      },

      changeNotice(notice) {
        this.state = this.state.set('notice', Immutable.fromJS(notice))
        this.trigger(this.state)
      },
    })

    const SyncedEmbed = React.createClass({
      displayName: 'SyncedEmbed',

      propTypes: {
        messageId: React.PropTypes.string,
        youtubeId: React.PropTypes.string,
        youtubeTime: React.PropTypes.number,
        startedAt: React.PropTypes.number,
        className: React.PropTypes.string,
      },

      shouldComponentUpdate(nextProps) {
        return nextProps.messageId !== this.props.messageId
      },

      render() {
        return (
          <Embed
            className={this.props.className}
            kind="youtube"
            autoplay="1"
            start={Math.max(0, Math.floor(Date.now() / 1000 - this.props.startedAt - clientTimeOffset)) + this.props.youtubeTime}
            youtube_id={this.props.youtubeId}
            messageId={this.props.messageId}
          />
        )
      },
    })

    const YouTubeTV = React.createClass({
      displayName: 'YouTubeTV',

      mixins: [
        Reflux.connect(TVStore, 'tv'),
        require('react-addons-pure-render-mixin'),
      ],

      render() {
        return (
          <SyncedEmbed
            className="youtube-tv"
            messageId={this.state.tv.getIn(['video', 'messageId'])}
            youtubeId={this.state.tv.getIn(['video', 'youtubeId'])}
            startedAt={this.state.tv.getIn(['video', 'time'])}
            youtubeTime={this.state.tv.getIn(['video', 'youtubeTime'])}
          />
        )
      },
    })

    const YouTubePane = React.createClass({
      displayName: 'YouTubePane',

      mixins: [
        Reflux.connect(TVStore, 'tv'),
        require('react-addons-pure-render-mixin'),
      ],

      render() {
        return (
          <div className="chat-pane-container youtube-pane">
            <div className="top-bar">
              <MessageText className="title" content={':notes: :tv: :notes: ' + this.state.tv.getIn(['video', 'title'])} />
            </div>
            <div className="aspect-wrapper">
              <YouTubeTV />
            </div>
            <MessageText className="notice-board" content={this.state.tv.getIn(['notice', 'content'])} />
          </div>
        )
      },
    })

    const parseYoutubeTime = function parseYoutubeTime(time) {
      const timeReg = /([0-9]+h)?([0-9]+m)?([0-9]+s?)?/
      const match = time.match(timeReg)
      if (!match) {
        return 0
      }
      const hours = parseInt(match[1] || 0, 10)
      const minutes = parseInt(match[2] || 0, 10)
      const seconds = parseInt(match[3] || 0, 10)
      return hours * 3600 + minutes * 60 + seconds
    }

    Heim.hook('thread-panes', () => {
      return <YouTubePane key="youtube-tv" />
    })

    Heim.hook('main-pane-top', function YouTubeTVInject() {
      return this.state.ui.thin ? <YouTubeTV key="youtube-tv" /> : null
    })

    Heim.chat.messagesChanged.listen(function onMessagesChanged(ids, state) {
      const candidates = Immutable.Seq(ids)
        .map(messageId => {
          const msg = state.messages.get(messageId)
          const valid = messageId !== '__root' && msg.get('content')
          return valid && msg
        })
        .filter(Boolean)

      const playRe = /!play [^?]*\?v=([-\w]+)(?:&t=([0-9hms]+))?/
      const video = candidates
        .map(msg => {
          const match = msg.get('content').match(playRe)
          return match && {
            time: msg.get('time'),
            messageId: msg.get('id'),
            youtubeId: match[1],
            youtubeTime: match[2] ? parseYoutubeTime(match[2]) : 0,
            title: msg.get('content'),
          }
        })
        .filter(Boolean)
        .sortBy(v => v.time)
        .last()

      if (video && video.time > TVStore.state.getIn(['video', 'time'])) {
        TVActions.changeVideo(video)
      }

      const noticeRe = /^!notice(\S*?)\s([^]*)$/
      const notices = candidates
        .map(msg => {
          const match = msg.get('content').match(noticeRe)
          return match && {
            id: msg.get('id'),
            time: msg.get('time'),
            display: !match[1].length,
            content: match[2],
          }
        })
        .filter(Boolean)
        .cacheResult()

      const noticeMaxSummaryLength = 80
      notices.forEach(notice => {
        const lines = notice.content.split('\n')
        let content = lines[0]
        if (content.length >= noticeMaxSummaryLength || lines.length > 1) {
          content = content.substr(0, noticeMaxSummaryLength) + '…'
        }
        state.messages.mergeNodes(notice.id, {
          content: '/me changed the notice to: "' + content + '"',
        })
      })

      const latestNotice = notices
        .filter(n => n.display)
        .sortBy(notice => notice.time)
        .last()

      if (latestNotice && latestNotice.time > TVStore.state.getIn(['notice', 'time'])) {
        TVActions.changeNotice(latestNotice)
      }
    })

    Heim.hook('page-bottom', () => {
      return (
        <style key="youtubetv-style" dangerouslySetInnerHTML={{__html: `
          .youtube-pane {
            z-index: 9;
          }

          .youtube-pane .title {
            width: 0;
          }

          .youtube-pane .aspect-wrapper {
            flex-shrink: 0;
            position: relative;
            width: 100%;
            box-shadow: 0 0 12px rgba(0, 0, 0, .25);
            z-index: 5;
          }

          .youtube-pane .aspect-wrapper:before {
            content: '';
            display: block;
            padding-top: 75%;
          }

          .youtube-pane .youtube-tv {
            position: absolute;
            top: 0;
            bottom: 0;
            left: 0;
            right: 0;
            width: 100%;
            height: 100%;
          }

          .youtube-tv {
            border: none;
            height: 60vmin;
          }

          .youtube-pane .notice-board {
            background: white;
            padding: 10px;
            overflow: auto;
            white-space: pre-wrap;
            flex: 1;
          }
        `}} />
      )
    })
  }

  if (roomName === 'adventure' || roomName === 'chess' || roomName === 'monospace') {
    Heim.hook('page-bottom', () => {
      return (
        <style key="adventure-style" dangerouslySetInnerHTML={{__html: `
          .messages-container, .messages-container input, .messages-container textarea {
            font-family: Droid Sans Mono, monospace;
          }
        `}} />
      )
    })

    Heim.chat.setRoomSettings({collapse: false})
  }

  if (uiwindow.location.hash.substr(1) === 'spooky') {
    Heim.hook('page-bottom', () => {
      return (
        <style key="spooky-style" dangerouslySetInnerHTML={{__html: `
          #ui {
            background: #281f3d;
          }

          .info-pane, .sidebar-pane, .top-bar {
            background: #2e293c;
          }

          .info-pane *, .top-bar *, .sidebar-pane * {
            color: darkorange !important;
          }

          .nick, .message-emote {
            color: black !important;
            -webkit-filter: saturate(2) brightness(.75);
            filter: saturate(2) brightness(.75);
          }

          .top-bar button {
            background: darkorange !important;
          }

          .main-pane .top-bar .hex {
            fill: darkorange !important;
          }

          .top-bar button .inner, .top-bar button .inner * {
            color: black !important;
          }

          .info-pane .thread-list-container {
            border: none !important;
          }

          .info-pane .thread-list-container:after {
            box-shadow: none;
          }

          .info-pane .thread-list-container .info:hover,
          .info-pane .thread-list-container .info.selected {
            background: black !important;
          }

          .info-pane .mode-selector {
            background: #444 !important;
          }

          .info-pane .mode-selector button .inner {
            filter: grayscale(1) invert(1);
            -webkit-filter: grayscale(1) invert(1);
          }

          .info-pane .mode-selector button.selected {
            background: darkorange !important;
          }

          .info-pane .mode-selector button.selected .inner {
            filter: grayscale(1) invert(1) brightness(0);
            -webkit-filter: grayscale(1) invert(1) brightness(0);
          }

          .info-pane .notification {
            background: none !important;
          }

          .messages .timestamp {
            color: darkorange !important;
          }

          .messages-content {
            background: none !important;
          }

          .messages-container, .youtube-pane .notice {
            background: linear-gradient(to bottom, #423553 40px, #443e5d) !important;
          }

          .timestamps-visible .messages-container {
            background:
              linear-gradient(to right, #262334 72px, transparent 72px),
              linear-gradient(to bottom, #423553 40px, #443e5d) !important;
          }

          .replies .entry:before {
            background-color: transparent !important;
          }

          .indent-line, .replies .entry:before, .expand-rest .inner:before {
            filter: invert(1) !important;
            -webkit-filter: invert(1) !important;
          }

          .expand-rest {
            color: darkorange !important;
          }

          .entry {
            background: rgba(0, 0, 0, .15) !important;
          }

          .entry .nick {
            background: rgba(255, 255, 255, .25) !important;
          }

          .entry input.nick {
            z-index: 10;
          }

          .entry-focus .entry, .expand-rest.focus-target {
            background: #90561f !important;
            border-bottom-color: darkorange !important;
            color: white !important;
          }

          .entry-focus .entry textarea, .line .message, .message-preview {
            color: white !important;
            text-shadow: 0 1px 1px black !important;
          }

          .message-emote {
            text-shadow: none !important;
          }

          .mention > .line .message {
            background: darkorange !important;
          }

          .message a {
            color: darkorange !important;
          }

          .expando:after {
            background: none !important;
          }

          .last-visit hr {
            border-color: darkorange !important;
          }

          .last-visit .label {
            color: darkorange !important;
          }

          .spinner {
            filter: invert(1) brightness(2);
            -webkit-filter: invert(1) brightness(2);
          }

          .new-count {
            color: #afa !important;
          }

          .youtube-pane .notice {
            color: white;
          }

          ::-webkit-scrollbar {
            width: 6px;
            background: none !important;
          }

          ::-webkit-scrollbar-track, ::-webkit-scrollbar {
            background: none !important;
          }

          ::-webkit-scrollbar-thumb {
            background: darkorange !important;
            border-radius: 2px;
          }

          ::-webkit-scrollbar-button {
            display: none;
          }

          @keyframes spooky {
            0% {
              transform: rotate(0deg) scale(1);
            }

            50% {
              transform: rotate(360deg) scale(1.5);
            }

            100% {
              transform: rotate(720deg) scale(1);
            }
          }

          .message .emoji-1f47b, .message .emoji-1f480, .message .emoji-1f383 {
            animation-name: spooky;
            animation-duration: 4s;
            animation-iteration-count: 3;
            animation-timing-function: linear;
            z-index: 1000;
          }
        `}} />
      )
    })
  }

  if (roomName === 'sandersforpresident') {
    Heim.hook('main-pane-top', function BernieBarInject() {
      const MessageText = require('./ui/MessageText').default
      return (
        <div key="sanders-top-bar" className="secondary-top-bar"><MessageText onlyEmoji content=":us:" /> Welcome to the <a href="https://reddit.com/r/sandersforpresident" target="_blank">/r/SandersForPresident</a> live chat! Please <a href="https://www.reddit.com/r/SandersForPresident/wiki/livechat" target="_blank">read our rules</a>.</div>
      )
    })

    Heim.hook('page-bottom', () => {
      return (
        <style key="sanders-style" dangerouslySetInnerHTML={{__html: `
          .top-bar {
            background: #327bbe;
          }

          .top-bar button:not(.manager-toggle) {
            background: rgba(255, 255, 255, .5) !important;
          }

          .secondary-top-bar {
            color: white;
            background: #193e60;
            padding: 10px 6px;
          }

          .secondary-top-bar a, .main-pane .top-bar .room .name {
            color: white;
          }

          .sidebar-pane {
            background: #f2f2f2;
          }

          .sidebar-pane h1 {
            color: #4d5763;
          },
        `}} />
      )
    })
  }

  if (uiwindow.location.hash.substr(1) === 'darcula') {
    Heim.hook('page-bottom', () => {
      return (
          <style key="darcula-style" dangerouslySetInnerHTML={{__html: `
          #ui {
            background: #281f3d;
          }

          .info-pane, .sidebar-pane, .top-bar {
            background: #4C5053;
          }

          .info-pane *, .top-bar *, .sidebar-pane * {
            color: #758076 !important;
          }

          .nick, .message-emote {
            color: black !important;
            -webkit-filter: saturate(2) brightness(.75);
            filter: saturate(2) brightness(.75);
          }

          .top-bar button {
            background: #4477B2 !important;
          }

          .top-bar button .inner, .top-bar button .inner * {
            color: black !important;
          }

          .info-pane .thread-list-container {
            border: none !important;
          }

          .info-pane .thread-list-container:after {
            box-shadow: none;
          }

          .info-pane .thread-list-container .info:hover,
          .info-pane .thread-list-container .info.selected {
            background: black !important;
          }

          .info-pane .mode-selector {
            background: #444 !important;
          }

          .info-pane .mode-selector button .inner {
            filter: grayscale(1) invert(1);
            -webkit-filter: grayscale(1) invert(1);
          }

          .info-pane .mode-selector button.selected {
            background: darkorange !important;
          }

          .info-pane .mode-selector button.selected .inner {
            filter: grayscale(1) invert(1) brightness(0);
            -webkit-filter: grayscale(1) invert(1) brightness(0);
          }

          .info-pane .notification {
            background: none !important;
          }

          .messages .timestamp {
            color: #849AAB !important;
          }

          .messages-content {
            background: none !important;
          }

          .messages-container, .youtube-pane .notice {
            background: linear-gradient(to bottom, #423553 40px, #443e5d) !important;
          }

          .timestamps-visible .messages-container {
            background:
             linear-gradient(to right, #3A3C3E 72px, transparent 72px),
             linear-gradient(to bottom, #1B1F20 40px, #1c2021) !important
          }

          .replies .entry:before {
            background-color: transparent !important;
          }

          .indent-line, .replies .entry:before, .expand-rest .inner:before {
            filter: invert(1) !important;
            -webkit-filter: invert(1) !important;
          }

          .expand-rest {
            color: #662E72 !important;
          }

          .entry {
            background: rgba(0, 0, 0, .15) !important;
          }

          .entry .nick {
            background: rgba(255, 255, 255, 1) !important;
          }

          .entry input.nick {
            z-index: 10;
          }

          .entry-focus .entry, .expand-rest.focus-target {
            background: #1e1f36 !important;
            border-bottom-color: #37385d !important;
            color: white !important;
          }

          .entry-focus .entry textarea, .line .message, .message-preview {
            color: white !important;
            text-shadow: 0 1px 1px black !important;
          }

          .message-emote {
            text-shadow: none !important;
          }

          .mention > .line .message {
            background: #4477B2 !important;
          }

          .message a {
            color: #662E72 !important;
          }

          .expando:after {
            background: none !important;
          }

          .last-visit hr {
            border-color: darkgray !important;
          }

          .last-visit .label {
            color: darkgray !important;
          }

          .spinner {
            filter: invert(1) brightness(2);
            -webkit-filter: invert(1) brightness(2);
          }

          .new-count {
            color: #afa !important;
          }

          .youtube-pane .notice {
            color: white;
          }

          ::-webkit-scrollbar {
            width: 6px;
            background: none !important;
          }

          ::-webkit-scrollbar-track, ::-webkit-scrollbar {
            background: none !important;
          }

          ::-webkit-scrollbar-thumb {
            background: darkgray !important;
            border-radius: 2px;
          }

          ::-webkit-scrollbar-button {
            display: none;
          }
        `}} />
      )
    })
  }

  if (roomName === 'xkcd') {
    Heim.hook('main-pane-top', () => {
      return (
        <div key="xkcd-top-bar" className="secondary-top-bar"><span className="motto" title="All problems are solvable by being thrown at with bots">Omnes qu&aelig;stiones solvuntur eis iactandis per machinis</span></div>
      )
    })

    Heim.hook('page-bottom', () => {
      return (
        <style key="xkcd-top-style" dangerouslySetInnerHTML={{__html: `
          .secondary-top-bar {
            color: black;
            background: white;
            padding: 0.25em;
            text-align: center;
            box-shadow: 0 0 8px rgba(0, 0, 0, 0.25);
            z-index: 10;
          }

          .motto {
            font-family: "Droid Serif", Georgia, serif;
            text-transform: uppercase;
            cursor: help;
          }

          .motto::before {
            content: "~ ";
          }
          .motto::after {
            content: " ~";
          }
        `}} />
      )
    })

    if (uiwindow.location.hash.substr(1) === 'spooky') {
      Heim.hook('page-bottom', () => {
        return (
          <style key="xkcd-top-spooky-style" dangerouslySetInnerHTML={{__html: `
            .secondary-top-bar {
              color: darkorange;
              background: #2e293c;
            }
          `}} />
        )
      })
    } else if (uiwindow.location.hash.substr(1) === 'darcula') {
      Heim.hook('page-bottom', () => {
        return (
          <style key="xkcd-top-darcula-style" dangerouslySetInnerHTML={{__html: `
            .secondary-top-bar {
              color: #758076;
              background: #4c5053;
            }
          `}} />
        )
      })
    }
  }

  const now = moment()
  if (now.month() === 11 && (now.date() === 13 || now.date() === 14)) {
    Heim.hook('page-bottom', () => {
      return (
        <style key="anniversary-style" dangerouslySetInnerHTML={{__html: `
          .messages-content {
            background-image: url(/static/anniversary.svg) !important;
            background-repeat: no-repeat !important;
            background-position: right 180px bottom 0 !important;
            background-size: 700px !important;
            background-attachment: fixed !important
          }

          @media (max-width: 650px) {
            .messages-content {
              background-size: 180px !important;
            }
          }
        `}} />
      )
    })
  }
}
