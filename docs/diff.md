# 对比inception


## 功能对比

功能  |  inception  |  goInception | 说明
------------ | :--------: | :--------: | ------------
审核   |  <div class="text-success">✓</div>    |   <div class="text-success">✓</div>     |   基本无差异
执行   |  <div class="text-success">✓</div>    |   <div class="text-success">✓</div>     |   基本无差异
pt-osc工具   |  <div class="text-success">✓</div>    |   <div class="text-success">✓</div>     |   基本无差异
gh-ost工具   |  <div class="text-error">✕</div>    |   <div class="text-success">✓</div>     |
备份   |  <div class="text-success">✓</div>    |   <div class="text-success">✓</div>     |   基本无差异
忽略警告   |  <div class="text-success">✓</div>    |   <div class="text-success">✓</div>     |   基本无差异
只读参数   |  <div class="text-success">✓</div>    |   <div class="text-error">✕</div>     |   goinception未提供
打印SQL语法树   |  <div class="text-success">✓</div>    |   <div class="text-success">✓</div>     |   inception的感觉更友好
DDL和DML拆分功能   |  <div class="text-success">✓</div>    |   <div class="text-success">✓</div>     |   goinception支持混合执行，不会影响回滚解析
执行部分后休眠   |  <div class="text-success">✓</div>    |   <div class="text-success">✓</div>     |   goinception支持执行指定条数后休眠
计算真实受影响行数   |  <div class="text-error">✕</div>    |   <div class="text-success">✓</div>     |
事务支持   |  <div class="text-error">✕</div>    |   <div class="text-success">✓</div>     |
SQL指纹功能   |  <div class="text-error">✕</div>    |   <div class="text-success">✓</div>     |   dml语句相似时，可以根据相同的指纹ID复用explain结果，以减少远端数据库explain操作，提高审核速度

## 速度

模块  |  inception  |  goInception | 说明
------------ | :--------: | :--------: | ------------
审核   |  <div class="progress"> <div class="rect left" style="width: 90px;"/> </div>    |   <div class="progress"> <div class="rect left" style="width: 80px;"/></div>    |   审核速度inception占优，优势微弱
执行   |  <div class="progress"> <div class="rect left" style="width: 90px;"/> </div>    |   <div class="progress"> <div class="rect left" style="width: 90px;"/></div>     |   执行速度相近
备份   |  <div class="progress"> <div class="rect left" style="width: 60px;"/> </div>    |   <div class="progress"> <div class="rect left" style="width: 90px;"/></div>    |   备份速度goinception领先(批量备份)，优势较大

## 上手和使用

分类  |  inception  |  goInception | 说明
------------ | :--------: | :--------: | ------------
快速部署   |  <div class="progress"> <div class="rect left" style="width: 30px;"/> </div>    |   <div class="progress"> <div class="rect left" style="width: 90px;"/></div>    |   goinception可使用二进制部署，下载即用
问题调试   |  <div class="progress"> <div class="rect left" style="width: 30px;"/> </div>    |   <div class="progress"> <div class="rect left" style="width: 90px;"/></div>     |   goinception有较多日志输出，便于问题快速定位
接口调用   |  限`python`,`c`,`c++`    |   实现了`mysql数据库驱动的语言`    |



## 部分优化说明


