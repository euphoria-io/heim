var React = require('react/addons')
var Reflux = require('reflux')
var Immutable = require('immutable')

var Embed = require('./ui/embed')


module.exports = function(roomName) {
  if (roomName == 'thedrawingroom' || roomName == 'lovenest') {
    Heim.hook('page-bottom', function() {
      return (
        <style key="drawingroom-style" dangerouslySetInnerHTML={{__html:`
          .chat {
            background: #333;
          }

          .chat .room .name {
            color: #222;
          }

          .chat time {
            opacity: .5;
          }

          .chat .room .privacy-level,
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
    Heim.hook('sidebar-bottom', function() {
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
            margin-top: 15px;
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

  if (roomName == 'mantodea') {
    var _ = require('lodash')

    var serverTime = Number.MAX_VALUE
    var cutoff = 7200
    var cutoffReached = false

    Heim.socket.store.listen(function(ev) {
      if (ev.status == 'receive' && ev.body.type == 'ping-event') {
        serverTime = ev.body.data.time
      }
    })

    Heim.hook('incoming-messages', function(messages) {
      _.remove(messages, function(msg) {
        var tooOld = msg.time < serverTime - cutoff
        cutoffReached = cutoffReached || tooOld
        return tooOld
      })
    })

    Heim.actions.loadMoreLogs.shouldEmit = function() {
      return !cutoffReached
    }
  }

  if (roomName == 'music' || roomName == 'youtube') {
    var clientTimeOffset = 0
    Heim.socket.store.listen(function(ev) {
      if (ev.status == 'receive' && ev.body.type == 'ping-event') {
        clientTimeOffset = Date.now() / 1000 - ev.body.data.time
      }
    })

    var TVActions = Reflux.createActions([
      'changeVideo',
    ])

    var TVStore = Reflux.createStore({
      listenables: [TVActions],

      init: function() {
        this.state = {
          youtubeId: null,
          time: 0,
        }
      },

      getInitialState: function() {
        return this.state
      },

      changeVideo: function(video) {
        this.state = video
        this.trigger(this.state)
      }
    })

    var YouTubeTV = React.createClass({
      displayName: 'YouTubeTV',

      mixins: [
        Reflux.connect(TVStore, 'tv'),
        React.addons.PureRenderMixin,
      ],

      render: function() {
        // jshint camelcase: false
        return <Embed
          className="youtube-tv"
          kind="youtube"
          autoplay="1"
          youtube_id={this.state.tv.youtubeId}
          start={Math.max(0, Math.floor(Date.now() / 1000 - this.state.tv.time - clientTimeOffset))}
        />
      }
    })

    Heim.hook('sidebar-top', function() {
      return <YouTubeTV key="youtube-tv" />
    })

    Heim.chat.messagesChanged.listen(function(ids, state) {
      var playRe = /!play [^?]*\?v=([-\w]+)/

      var video = Immutable.Seq(ids)
        .map(id => state.messages.get(id))
        .map(msg => {
          if (msg.get('id') == '__root') {
            return
          }
          var match = msg.get('content').match(playRe)
          return match && {time: msg.get('time'), youtubeId: match[1]}
        })
        .filter(Boolean)
        .sortBy(msg => msg.time)
        .last()

      if (video && video.time > TVStore.state.time) {
        TVActions.changeVideo(video)
      }
    })

    Heim.hook('page-bottom', function() {
      return (
        <style key="youtubetv-style" dangerouslySetInnerHTML={{__html:`
          .youtube-tv {
            width: 240px;
            height: 180px;
            margin-bottom: 15px;
            border: none;
          }

          .chat .user-list {
            max-height: calc(80vh - 195px);
          }

          @media (min-width: 500px) {
            .chat .sidebar {
              width: 240px;
            }

            .chat .messages-container .messages .line {
              margin-right: 245px;
            }
          }

          @media (min-width: 920px) {
            .youtube-tv {
              width: 360px;
              height: 270px;
            }

            .chat .user-list {
              max-height: calc(80vh - 285px);
            }

            .chat .sidebar {
              width: 360px;
            }

            .chat .messages-container .messages .line {
              margin-right: 365px;
            }
          }
        `}} />
      )
    })
  }

  if (roomName == 'adventure') {
    Heim.hook('page-bottom', function() {
      return (
        <style key="adventure-style" dangerouslySetInnerHTML={{__html:`
          .messages-container, .messages-container input, .messages-container textarea, .sidebar {
            font-family: Droid Sans Mono, monospace;
          }
        `}} />
      )
    })

    Heim.chat.setRoomSettings({collapse: false})
  }
}
