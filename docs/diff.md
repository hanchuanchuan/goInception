# Compared with inception


## Function comparison:

Function  |  inception  |  goInception | Description
------------ | :--------: | :--------: | ------------
Audit   |  <div class="text-success">✓</div>    |   <div class="text-success">✓</div>     |  Basically no difference
Execute   |  <div class="text-success">✓</div>    |   <div class="text-success">✓</div>     |   Basically no difference
pt-osc tool   |  <div class="text-success">✓</div>    |   <div class="text-success">✓</div>     |   Basically no difference
gh-ost tool  |  <div class="text-error">✕</div>    |   <div class="text-success">✓</div>     |
Backup   |  <div class="text-success">✓</div>    |   <div class="text-success">✓</div>     |   Basically no difference
Ignore warning parameters   |  <div class="text-success">✓</div>    |   <div class="text-success">✓</div>     |   Basically no difference
Read-only parameter   |  <div class="text-success">✓</div>    |   <div class="text-error">✕</div>     |   goinception not provided
Syntax tree   |  <div class="text-success">✓</div>    |   <div class="text-success">✓</div>     |   The syntax tree of inception is more friendly
Split function of DDL and DML   |  <div class="text-success">✓</div>    |   <div class="text-success">✓</div>     |   goinception supports mixed execution and will not affect rollback analysis
Hibernate after executing a batch   |  <div class="text-success">✓</div>    |   <div class="text-success">✓</div>     |   goinception supports sleeping after executing the specified number
Number of affected rows   |  <div class="text-error">✕</div>    |   <div class="text-success">✓</div>     | goinception supports the calculation of the actual number of affected rows
Transaction   |  <div class="text-error">✕</div>    |   <div class="text-success">✓</div>     |
SQL fingerprint   |  <div class="text-error">✕</div>    |   <div class="text-success">✓</div>     |   When the dml statements are similar, the explain results can be reused according to the same fingerprint ID to reduce the remote database explain operations and improve the audit speed

## Speed comparison

Stage  |  inception  |  goInception | Description
------------ | :--------: | :--------: | ------------
Audit   |  <div class="progress"> <div class="rect left" style="width: 90px;"/> </div>    |   <div class="progress"> <div class="rect left" style="width: 80px;"/></div>    |   Slightly better review speed inception
Execute   |  <div class="progress"> <div class="rect left" style="width: 90px;"/> </div>    |   <div class="progress"> <div class="rect left" style="width: 90px;"/></div>     |   Similar execution speed
Backup   |  <div class="progress"> <div class="rect left" style="width: 20px;"/> </div>    |   <div class="progress"> <div class="rect left" style="width: 90px;"/></div>    |   Goinception leads in backup speed (batch backup), which has a greater advantage

## Difficult to get started

operating  |  inception  |  goInception | Description
------------ | :--------: | :--------: | ------------
Rapid deployment   |  <div class="progress"> <div class="rect left" style="width: 30px;"/> </div>    |   <div class="progress"> <div class="rect left" style="width: 90px;"/></div>    |   Goinception can use binary deployment, download and use
Problem debugging   |  <div class="progress"> <div class="rect left" style="width: 30px;"/> </div>    |   <div class="progress"> <div class="rect left" style="width: 90px;"/></div>     |   Goinception has a lot of log output, which is easy to locate the problem quickly
Interface call   |  Limited to `python`,`c`,`c++`    |   As long as the language of `mysql database driver` is implemented, the call is supported    |




