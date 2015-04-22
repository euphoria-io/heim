var _ = require('lodash')
var Immutable = require('immutable')
var EventEmitter = require('eventemitter3')


function Tree(sortProp, updateFunc) {
  this.sortProp = sortProp
  this.updateFunc = updateFunc || _.noop
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

  _add: function(entry, _changed, _needsSort) {
    if (entry.parent === undefined) {
      entry.parent = '__root'
    }

    var newId = entry.id
    var newNode = this._node(entry)
    var parentId = entry.parent

    if (parentId) {
      var parentNode = this.index[parentId]
      if (parentNode) {
        var siblings = parentNode.get('children')
        var canAppend = siblings.size === 0 || entry[this.sortProp] >= this.index[siblings.last()].get(this.sortProp)
        if (canAppend) {
          // avoiding a re-sort can save a lot of time with large child lists
          this.index[parentId] = parentNode.set('children', siblings.delete(newId).add(newId))
          if (!Immutable.is(parentNode, this.index[parentId])) {
            _changed[parentId] = _changed[parentId] || parentNode
          }
        } else {
          this.index[parentId] = parentNode.set('children', siblings.add(newId))
          _needsSort[parentId] = true
          _changed[parentId] = _changed[parentId] || parentNode
        }
      } else {
        // create unreachable orphan parent
        parentNode = this.index[parentId] = this._node().set('id', parentId).mergeIn(['children'], [newId])
        _changed[parentId] = _changed[parentId] || true
        this.size++
        _needsSort[parentId] = true
      }
    }

    if (_.has(this.index, newId)) {
      var oldNode = this.index[newId]

      var oldParentId = oldNode.get('parent')
      var oldParent = oldParentId && oldParentId != parentId && this.index[oldParentId]
      if (oldParent) {
        this.index[oldParentId] = oldParent.set('children', oldParent.get('children').remove(newId))
        _changed[oldParentId] = _changed[oldParentId] || oldParent
      }

      // merge in orphans
      newNode = oldNode.mergeDeep(newNode)
      if (!_changed[newId] && !Immutable.is(oldNode, newNode)) {
        _changed[newId] = oldNode
      }
    } else {
      _changed[newId] = _changed[newId] || true
      this.size++
    }

    if (_changed[newId]) {
      this.index[newId] = newNode

      if (entry[this.sortProp] > this._lastValue) {
        this._lastId = newId
        this._lastValue = entry[this.sortProp]
      }
    }
  },

  _updateChanged: function(changed) {
    if (!_.isEmpty(changed)) {
      var updateNode = node => {
        var nodeId = node.get('id')
        var existing = this.index[nodeId]
        if (!Immutable.is(node, existing)) {
          changed[nodeId] = changed[nodeId] || existing
          this.index[nodeId] = node
        }
        return node
      }
      this.updateFunc(changed, updateNode)

      var changedIds = _.keys(changed)
      _.each(changedIds, id => {
        this.changes.emit(id, this.index[id])
      })
      this.changes.emit('__all', changedIds)
    }
  },

  add: function(entries, _sorted) {
    if (!_.isArray(entries)) {
      entries = [entries]
    }

    var _changed = {}
    var _needsSort = {}

    _.each(entries, entry => {
      this._add(entry, _changed, _needsSort)
    })

    if (!_sorted) {
      _.each(_needsSort, (x, id) => {
        var resorted = this.index[id].get('children').sortBy(childId => {
          return this.index[childId].get(this.sortProp)
        })
        this.index[id] = this.index[id].set('children', resorted)

        // if parent now matches the original one, no need to emit the change
        if (Immutable.is(_changed[id], this.index[id])) {
          delete _changed[id]
        }
      })
    }

    this._updateChanged(_changed)
  },

  mergeNodes: function(ids, data) {
    if (!_.isArray(ids)) {
      ids = [ids]
    }

    var changed = {}
    _.each(ids, id => {
      var old = this.index[id]
      this.index[id] = this.index[id].mergeDeep(data)
      if (old != this.index[id]) {
        changed[id] = old
      }
    })
    this._updateChanged(changed)
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

  lazyMapDFS: function(visit, thisArg, nodeId, depth) {
    if (!nodeId) {
      nodeId = '__root'
    }

    if (depth === undefined) {
      depth = 0
    }

    var node = this.index[nodeId]
    var children = node.get('children').toSeq().map((childId) =>
      this.lazyMapDFS(visit, thisArg, childId, depth + 1)
    )

    return visit.call(thisArg, node, children, depth)
  },

  mapDFS: function(visit, thisArg, nodeId, depth) {
    var visitStrict = (node, children, depth) => visit.call(thisArg, node, children.cacheResult(), depth)
    return this.lazyMapDFS(visitStrict, thisArg, nodeId, depth)
  },

  iterChildrenOf: function(nodeId) {
    var children = this.index[nodeId].get('children').toJS()
    return {
      next: () => {
        var childId = children.shift()
        if (!childId) {
          return {done: true}
        } else {
          return {done: false, value: this.index[childId]}
        }
      },
    }
  },

  iterAncestorsOf: function(nodeId) {
    return {
      next: () => {
        nodeId = this.index[nodeId].get('parent')
        if (!nodeId) {
          return {done: true}
        } else {
          return {done: false, value: this.index[nodeId]}
        }
      },
    }
  },

  last: function() {
    return this.index[this._lastId]
  },

  clone: function() {
    var newTree = new Tree(this.sortProp, this.updateFunc)
    newTree.index = _.clone(this.index)
    newTree._lastId = this._lastId
    newTree._lastValue = this._lastValue
    newTree.size = this.size
    return newTree
  },
})

module.exports = Tree
