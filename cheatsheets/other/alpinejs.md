---
title: Alpine.js
category: JavaScript library
---

# Intro

[Alpine.js](https://alpinejs.dev/) is a minimalist, reactive JavaScript framework.

To include Alpine.js in your HTML:
```html
<script src="//unpkg.com/alpinejs" defer></script>

<div x-data="{ open: false }">
    <button @click="open = true">Expand</button>
 
    <span x-show="open">
      Content...
    </span>
</div>
```

# Attributes

## x-data

Declare a new Alpine component and its data for a block of HTML

```html
<div x-data="{ open: false }">
    ...
</div>
```

## x-bind

Dynamically set HTML attributes on an element

```html
<div x-bind:class="! open ? 'hidden' : ''">
  ...
</div>
```

## x-on

```html
<button x-on:click="open = ! open">
  Toggle
</button>
```

## x-text

Set the text content of an element

```html

<div>
  Copyright ©
 
  <span x-text="new Date().getFullYear()"></span>
</div>
```

## x-html

Set the inner HTML of an element

```html
<div x-html="(await axios.get('/some/html/partial')).data">
  ...
</div>
```

## x-model

Synchronize a piece of data with an input element

```html
<div x-data="{ search: '' }">
  <input type="text" x-model="search">
 
  Searching for: <span x-text="search"></span>
</div>```

## x-show

Toggle the visibility of an element.

```html
<div x-show="open">
  ...
</div>
```


## x-transition

Transition an element in and out using CSS transitions

```html
<div x-show="open" x-transition>
  ...
</div>
```

## x-for

Repeat a block of HTML based on a data set

```html
<template x-for="post in posts">
  <h2 x-text="post.title"></h2>
</template>
```

## x-if

Conditionally add/remove a block of HTML from the page entirely.

Only use on `<template>`, use x-show for HTML elements.

```html
<template x-if="open">
  <div>...</div>
</template>
```

## x-init

Run code when an element is initialized by Alpine

```html
<div x-init="date = new Date()"></div>
```

## x-effect

Execute a script each time one if its dependancies change

```html
<div x-effect="console.log('Count is '+count)"></div>
```

## x-ref

Reference elements directly by their specified keys using the $refs magic property

```html
<input type="text" x-ref="content">
 
<button x-on:click="navigator.clipboard.writeText($refs.content.value)">
  Copy
</button>
```

## x-cloak

Hide a block of HTML until after Alpine is finished initializing its contents

```html
<div x-cloak>
  ...
</div>
```

## x-ignore

Prevent a block of HTML from being initialized by Alpine

```html
<div x-ignore>
  ...
</div>
```

# Properties

## $store

Access a global store registered using Alpine.store(...)

```html
<h1 x-text="$store.site.title"></h1>
```

## $el

Reference the current DOM element

```html
<div x-init="new Pikaday($el)"></div>
```

## $dispatch

Dispatch a custom browser event from the current element

```html
<div x-on:notify="...">
  <button x-on:click="$dispatch('notify')">...</button>
</div>
```

## $watch

Watch a piece of data and run the provided callback anytime it changes

```html
<div x-init="$watch('count', value => {
  console.log('count is ' + value))"
}">...</div>
```

## $refs

Reference an element by key (specified using x-ref)

```html
<div x-init="$refs.button.remove()">
  <button x-ref="button">Remove Me</button>
</div>
```

## $nextTick

Wait until the next "tick" (browser paint) to run a bit of code

```html
<div
  x-text="count"
  x-text="$nextTick(() => {"
    console.log('count is ' + $el.textContent)
  })
>...</div>
```


# Methods

## Alpine.data

Reuse a data object and reference it using x-data

```html
<div x-data="dropdown">
  ...
</div>
```

```js
 
Alpine.data('dropdown', () => ({
  open: false,
 
  toggle() { 
    this.open = ! this.open
  }
}))
```

## Alpine.store

Declare a piece of global, reactive, data that can be accessed from anywhere using $store

```html
<button @click="$store.notifications.notify('...')">
  Notify
</button>
```

```js
Alpine.store('notifications', {
  items: [],
 
  notify(message) { 
    this.items.push(message)
  }
})
```
