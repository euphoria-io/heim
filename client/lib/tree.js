var _ = require('lodash')
var Immutable = require('immutable')
var EventEmitter = require('eventemitter3')


function Tree(sortProp) {
  this.sortProp = sortProp
  this.changes = new EventEmitter()
  this.reset()
}

_.extend(Tree.prototype, {
  _node: function(entry) {
    return Immutable.fromJS(entry || {})
      .merge({
        'children': Immutable.OrderedSet(),
      })
  },

  _add: function(entry) {
    if (!entry.parent) {
      entry.parent = '__root'
    }

    var changed = false
    var newNode = this._node(entry)
    var newId = newNode.get('id')

    var parentId = newNode.get('parent')
    var parentNode = this.index[parentId]
    if (parentNode) {
      parentNode = this.index[parentId] = parentNode.mergeIn(['children'], [newId])
    } else {
      // create unreachable orphan parent
      parentNode = this.index[parentId] = this._node().mergeIn(['children'], [newId])
      this.size++
    }

    if (_.has(this.index, newId)) {
      // merge in orphans
      var oldNode = this.index[newId]
      newNode = oldNode.mergeDeep(newNode)
      changed = !Immutable.is(oldNode, newNode)
    } else {
      changed = true
      this.size++
    }

    if (changed) {
      this.index[newId] = newNode

      if (entry[this.sortProp] > this._lastValue) {
        this._lastId = newId
        this._lastValue = entry[this.sortProp]
      }
    }

    return changed
  },

  add: function(entries, _sorted) {
    if (!_.isArray(entries)) {
      entries = [entries]
    }

    var parents = {}
    _.each(entries, function(entry) {
      var parentId = entry.parent || '__root'
      if (_.has(this.index, parentId)) {
        parents[parentId] = this.index[parentId]
      }
    }, this)

    var changed = {}
    _.each(entries, function(entry) {
      if (this._add(entry)) {
        changed[entry.id] = true
      }
    }, this)

    if (!_sorted) {
      _.each(parents, function(oldNode, id) {
        var resorted = this.index[id].get('children').sortBy(function(childId) {
          return this.index[childId].get(this.sortProp)
        }.bind(this))
        this.index[id] = this.index[id].set('children', resorted)
      }, this)
    }

    _.each(parents, function(oldNode, id) {
      if (!Immutable.is(oldNode, this.index[id])) {
        changed[id] = true
      }
    }, this)

    if (!_.isEmpty(changed)){
      _.each(changed, function(item, id) {
        this.changes.emit(id, this.index[id])
      }, this)

      this.changes.emit('__all', _.keys(changed))
    }
  },

  mergeNode: function(id, data) {
    var old = this.index[id]
    this.index[id] = this.index[id].mergeDeep(data)
    if (old != this.index[id]) {
      this.changes.emit(id, this.index[id])
      this.changes.emit('__all', [id])
    }
  },

  reset: function(entries) {
    this.index = {}
    this.index.__root = this._node({id: '__root'})
    this._lastId = null
    this._lastValue = null
    this.size = 0

    if (entries) {
      this.add(entries, true)
    } else {
      this.changes.emit('__root', this.index.__root)
      this.changes.emit('__all', ['__root'])
    }
    return this
  },

  get: function(id) {
    return this.index[id]
  },

  mapDFS: function(visit, thisArg, nodeId, depth) {
    if (!nodeId) {
      nodeId = '__root'
    }

    if (depth === undefined) {
      depth = 0
    }

    var node = this.index[nodeId]
    var children = node.get('children').toSeq().map(function(childId) {
      return this.mapDFS(visit, thisArg, childId, depth + 1)
    }, this)

    return visit.call(thisArg, node, children, depth)
  },

  last: function() {
    return this.index[this._lastId]
  }
})

module.exports = Tree
