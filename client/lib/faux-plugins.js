var React = require('react/addons')
var Reflux = require('reflux')
var Immutable = require('immutable')


// allow var redeclaration for import dupes
// jshint -W004

module.exports = function(roomName) {
  if (roomName == 'thedrawingroom' || roomName == 'lovenest' || roomName == 'has') {
    Heim.hook('page-bottom', function() {
      return (
        <style key="drawingroom-style" dangerouslySetInnerHTML={{__html:`
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

  if (roomName == 'space') {
    var Embed = require('./ui/embed')

    Heim.hook('main-sidebar', function() {
      return (
        <div key="norman" className="norman">
          <p>norman</p>
          <Embed kind="img" url="//i.imgur.com/UKbitCO.jpg" />
        </div>
      )
    })

    Heim.hook('page-bottom', function() {
      return (
        <style key="norman-style" dangerouslySetInnerHTML={{__html:`
          .norman {
            text-align: right;
            opacity: .5;
          }

          .norman, .norman .embed {
            transition: transform .15s ease;
          }

          .norman:hover {
            opacity: 1;
          }

          .norman:hover .embed {
            transform: translate(-50%, 50%) scale(2);
          }

          .norman p {
            margin: 0;
            font-size: 12px;
          }

          .norman .embed {
            width: 0;
            height: 100px;
            border: none;
          }
        `}} />
      )
    })
  }

  if (roomName == 'music' || roomName == 'youtube') {
    var Embed = require('./ui/embed')
    var MessageText = require('./ui/message-text')
    var ChatPane = require('./ui/chat-pane')

    var clientTimeOffset = 0
    Heim.socket.store.listen(function(ev) {
      if (ev.status == 'receive' && ev.body.type == 'ping-event') {
        clientTimeOffset = Date.now() / 1000 - ev.body.data.time
      }
    })

    var TVActions = Reflux.createActions([
      'changeVideo',
    ])

    var tvPane = Heim.ui.createCustomPane('youtube-tv')

    var TVStore = Reflux.createStore({
      listenables: [
        TVActions,
        {chatChange: Heim.chat.store},
      ],

      init: function() {
        this.state = {
          time: 0,
          messageId: null,
          youtubeId: null,
        }
      },

      getInitialState: function() {
        return this.state
      },

      chatChange: function(state) {
        this.chatState = state
      },

      changeVideo: function(video) {
        // FIXME: abstract this process more cleanly
        var oldMessageId = this.state.messageId
        if (oldMessageId) {
          this.chatState.messages.mergeNodes(oldMessageId, {_inCustomPane: false, _collapseCaption: null})
        }
        this.state = video
        tvPane.store._reset({rootId: video.messageId})
        tvPane.focusMessage(video.messageId)
        this.chatState.messages.mergeNodes(video.messageId, {_inCustomPane: 'youtube-tv', _collapseCaption: 'playing'})
        this.trigger(this.state)
      }
    })

    var YouTubePane = React.createClass({
      displayName: 'YouTubePane',

      mixins: [
        Reflux.connect(TVStore, 'tv'),
        React.addons.PureRenderMixin,
      ],

      render: function() {
        // jshint camelcase: false
        return (
          <div className="chat-pane-container youtube-pane">
            <div className="top-bar">
              <MessageText className="title" content=":notes: :tv: :notes:" />
            </div>
            <div className="aspect-wrapper">
              <Embed
                className="youtube-tv"
                kind="youtube"
                autoplay="1"
                youtube_id={this.state.tv.youtubeId}
                start={Math.max(0, Math.floor(Date.now() / 1000 - this.state.tv.time - clientTimeOffset))}
              />
            </div>
            {this.state.tv.youtubeId && <ChatPane pane={tvPane} showParent={true} showAllReplies={true} />}
          </div>
        )
      }
    })

    Heim.hook('thread-panes', function() {
      return <YouTubePane key="youtube-tv" />
    })

    Heim.chat.messagesChanged.listen(function(ids, state) {
      var playRe = /!play [^?]*\?v=([-\w]+)/

      var video = Immutable.Seq(ids)
        .map(messageId => {
          var msg = state.messages.get(messageId)
          if (messageId == '__root' || !msg.get('content')) {
            return
          }
          var match = msg.get('content').match(playRe)
          return match && {
            time: msg.get('time'),
            messageId: messageId,
            youtubeId: match[1],
          }
        })
        .filter(Boolean)
        .sortBy(video => video.time)
        .last()

      if (video && video.time > TVStore.state.time) {
        TVActions.changeVideo(video)
      }
    })

    Heim.hook('page-bottom', function() {
      return (
        <style key="youtubetv-style" dangerouslySetInnerHTML={{__html:`
          .youtube-pane {
            z-index: 9;
          }

          .youtube-pane .aspect-wrapper {
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
            border: none;
          }
        `}} />
      )
    })
  }

  if (roomName == 'adventure' || roomName == 'chess') {
    Heim.hook('page-bottom', function() {
      return (
        <style key="adventure-style" dangerouslySetInnerHTML={{__html:`
          .messages-container, .messages-container input, .messages-container textarea {
            font-family: Droid Sans Mono, monospace;
          }
        `}} />
      )
    })

    Heim.chat.setRoomSettings({collapse: false})
  }
}
