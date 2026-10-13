<script setup>
import { ref, watch } from 'vue'
import { EditorContent, useEditor } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import Image from '@tiptap/extension-image'
import Placeholder from '@tiptap/extension-placeholder'

const props = defineProps({
  modelValue: { type: String, default: '' },
})
const emit = defineEmits(['update:modelValue'])

const linkOpen = ref(false)
const linkURL = ref('')
const linkError = ref('')

function storedHTML(html) {
  const value = String(html || '').trim()
  return value === '<p></p>' ? '' : value
}

const editor = useEditor({
  content: props.modelValue || '<p></p>',
  extensions: [
    StarterKit.configure({
      heading: { levels: [2, 3] },
      code: false,
      codeBlock: false,
      link: {
        openOnClick: false,
        autolink: true,
        defaultProtocol: 'https',
        isAllowedUri: (url, { defaultValidate }) => {
          if (url.startsWith('/') && !url.startsWith('//') && !url.includes(' ')) return true
          return defaultValidate(url)
        },
      },
    }),
    Image.configure({ allowBase64: false }),
    Placeholder.configure({ placeholder: '在这里写正文，选中文字后可以加粗、加标题或加链接' }),
  ],
  editorProps: {
    attributes: { class: 'prose rich-body' },
  },
  onUpdate: ({ editor: current }) => {
    emit('update:modelValue', storedHTML(current.getHTML()))
  },
})

watch([editor, () => props.modelValue], () => {
  const current = editor.value
  if (!current) return
  const incoming = props.modelValue || ''
  if (storedHTML(current.getHTML()) === incoming) return
  current.commands.setContent(incoming || '<p></p>', { emitUpdate: false })
})

function run(build) {
  const current = editor.value
  if (!current) return
  build(current.chain().focus()).run()
}

function openLink() {
  linkError.value = ''
  linkURL.value = editor.value?.getAttributes('link').href || 'https://'
  linkOpen.value = true
}

function normalizeLink(raw) {
  const url = String(raw || '').trim()
  if (!url) return ''
  if (url.startsWith('/') || /^https?:\/\//i.test(url) || /^mailto:/i.test(url)) return url
  if (/^[a-z0-9.-]+\.[a-z]{2,}([/?#].*)?$/i.test(url)) return `https://${url}`
  return ''
}

function applyLink() {
  linkError.value = ''
  const url = normalizeLink(linkURL.value)
  if (!linkURL.value.trim()) {
    run((chain) => chain.extendMarkRange('link').unsetLink())
    linkOpen.value = false
    return
  }
  if (!url) {
    linkError.value = '链接需要是网址、邮箱，或站内路径'
    return
  }
  run((chain) => chain.extendMarkRange('link').setLink({ href: url }))
  linkOpen.value = false
}

function insertAsset(asset) {
  const current = editor.value
  if (!current || !asset?.url) return
  if (asset.mime === 'application/pdf') {
    current.chain().focus().insertContent({
      type: 'text',
      text: asset.name || '附件',
      marks: [{ type: 'link', attrs: { href: asset.url } }],
    }).run()
    return
  }
  current.chain().focus().setImage({ src: asset.url, alt: asset.name || '' }).run()
}

function html() {
  return storedHTML(editor.value?.getHTML() || props.modelValue)
}

defineExpose({ insertAsset, html })
</script>

<template>
  <div class="rich">
    <div class="rich-tools">
      <button class="btn small" type="button" :class="{ on: editor?.isActive('heading', { level: 2 }) }" @click="run((chain) => chain.toggleHeading({ level: 2 }))">标题</button>
      <button class="btn small" type="button" :class="{ on: editor?.isActive('heading', { level: 3 }) }" @click="run((chain) => chain.toggleHeading({ level: 3 }))">小标题</button>
      <button class="btn small" type="button" :class="{ on: editor?.isActive('bold') }" @click="run((chain) => chain.toggleBold())">加粗</button>
      <button class="btn small" type="button" :class="{ on: editor?.isActive('italic') }" @click="run((chain) => chain.toggleItalic())">斜体</button>
      <button class="btn small" type="button" :class="{ on: editor?.isActive('underline') }" @click="run((chain) => chain.toggleUnderline())">下划线</button>
      <button class="btn small" type="button" :class="{ on: editor?.isActive('strike') }" @click="run((chain) => chain.toggleStrike())">删除线</button>
      <button class="btn small" type="button" :class="{ on: editor?.isActive('bulletList') }" @click="run((chain) => chain.toggleBulletList())">列表</button>
      <button class="btn small" type="button" :class="{ on: editor?.isActive('orderedList') }" @click="run((chain) => chain.toggleOrderedList())">编号</button>
      <button class="btn small" type="button" :class="{ on: editor?.isActive('blockquote') }" @click="run((chain) => chain.toggleBlockquote())">引用</button>
      <button class="btn small" type="button" :class="{ on: editor?.isActive('link') }" @click="openLink">链接</button>
      <button class="btn small" type="button" @click="run((chain) => chain.setHorizontalRule())">分隔线</button>
      <slot name="tools" />
      <button class="btn small" type="button" @click="run((chain) => chain.unsetAllMarks().clearNodes())">清除格式</button>
    </div>
    <form v-if="linkOpen" class="rich-link" @submit.prevent="applyLink">
      <label class="field grow">
        <span>链接地址</span>
        <input v-model="linkURL" placeholder="https:// 或站内路径" />
      </label>
      <button class="btn small primary" type="submit">使用链接</button>
      <button class="btn small" type="button" @click="linkURL = ''; applyLink()">去掉链接</button>
      <button class="btn small" type="button" @click="linkOpen = false">取消</button>
      <p v-if="linkError" class="msg error">{{ linkError }}</p>
    </form>
    <EditorContent :editor="editor" />
  </div>
</template>
