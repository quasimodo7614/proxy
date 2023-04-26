# api 
## 发起文字聊天
### 路径: /chat
### 入参:

```json
{
   "msg":[
      {"role":"user","content":"我想查询一下我的订单状态"}, 
      {"role":"assistant","content":"请问您的订单号是多少？"}
   ]
}
```

### 出参:

```json
{"cont":""}
```

## 发起生成图片
### 路径: /image
### 入参:

```json
{
   "msg":"帮我画一只猫"
}
```

### 出参:
```json
{
   "data": [
      {"url":"xxx"}
   ]
}
```