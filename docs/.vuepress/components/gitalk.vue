<template>
  <div class="gitalk-container">
    <div id="gitalk-container"></div>
  </div>
</template>
<script>

import 'gitalk/dist/gitalk.css'
import Gitalk from 'gitalk'

export default {
  name: 'comment',
  data() {
    return {};
  },
  mounted() {
    const commentConfig = {
        clientID: '064577e7ed0d7ec4d5bc',
        clientSecret: 'cd00e6b2827490873e85d36f8e64b4721fb7c690',
        repo: 'goInception',
        owner: 'hanchuanchuan',
        admin: ['hanchuanchuan'],
        // id 用于当前页面的唯一标识，一般来讲 pathname 足够了，
        // 但是如果你的 pathname 超过 50 个字符，GitHub 将不会成功创建 issue，此情况可以考虑给每个页面生成 hash 值的方法.
        id: location.pathname,
        distractionFreeMode: false,
    };

    const gitalk = new Gitalk(commentConfig);
    gitalk.render('gitalk-container');
  },
};
</script>
