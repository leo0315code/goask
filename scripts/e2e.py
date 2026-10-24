#!/usr/bin/env python3
"""GoAsk 端到端冒烟测试（需本地服务已启动：make run 或 docker compose up）。

用法：
    python3 scripts/e2e.py            # 默认 http://127.0.0.1:8080
    BASE=http://host:port python3 scripts/e2e.py

注意：会在数据库中创建随机后缀的测试用户与内容，请勿对生产库执行。
"""
import json
import os
import random
import string
import urllib.error
import urllib.request

BASE = os.environ.get("BASE", "http://127.0.0.1:8080") + "/api/v1"
SUFFIX = "".join(random.choices(string.ascii_lowercase + string.digits, k=6))

passed, failed = 0, 0


def call(method, path, body=None, token=None):
    req = urllib.request.Request(BASE + path, method=method)
    req.add_header("Content-Type", "application/json")
    if token:
        req.add_header("Authorization", "Bearer " + token)
    data = json.dumps(body).encode() if body is not None else None
    try:
        with urllib.request.urlopen(req, data) as r:
            return r.status, json.load(r)
    except urllib.error.HTTPError as e:
        return e.code, json.load(e)


def check(name, cond, detail=""):
    global passed, failed
    if cond:
        passed += 1
        print(f"PASS  {name}")
    else:
        failed += 1
        print(f"FAIL  {name}  {detail}")


import re


def solve_captcha(question):
    m = re.search(r"(\d+)\s*([+\-])\s*(\d+)", question)
    a, op, b = int(m.group(1)), m.group(2), int(m.group(3))
    return str(a + b if op == "+" else a - b)


def register(name):
    # 注册需要算术验证码：领取 → 解题 → 注册；失败再尝试登录（重复运行幂等）
    _, cap = call("GET", "/auth/captcha")
    cid, q = cap["data"]["captcha_id"], cap["data"]["svg"]
    q = "".join(re.findall(r">([^<]+)<", q))
    s, r = call("POST", "/auth/register", {
        "username": name, "email": f"{name}@e2e.test", "password": "secret123",
        "captcha_id": cid, "captcha_code": solve_captcha(q),
    })
    if s == 200 and r["code"] == 0:
        return r["data"]["token"]
    s, r = call("POST", "/auth/login", {"account": name, "password": "secret123"})
    if not (s == 200 and r["code"] == 0):
        raise RuntimeError(f"注册/登录失败: {r}")
    return r["data"]["token"]


U1, U2, U3 = f"e2e_a_{SUFFIX}", f"e2e_b_{SUFFIX}", f"e2e_c_{SUFFIX}"
T1, T2, T3 = register(U1), register(U2), register(U3)

# ---- 注册 / 登录 ----
s, r = call("POST", "/auth/login", {"account": U1, "password": "secret123"})
check("登录", s == 200 and r["code"] == 0, r)

# ---- 问题 ----
s, r = call("POST", "/questions", {"title": f"E2E 冒烟测试的问题标题（{SUFFIX}）", "content": "正文内容，用于端到端验证。", "tags": ["go", "e2e"]}, T1)
check("提问（含标签）", s == 200 and r["code"] == 0 and [t["name"] for t in r["data"]["tags"]] == ["go", "e2e"], r)
qid = r["data"]["id"]
s, r = call("GET", f"/questions/{qid}")
check("详情可访问", s == 200 and r["code"] == 0 and r["data"]["title"].startswith("E2E"), r)

# ---- 回答 / 采纳 ----
s, r = call("POST", f"/questions/{qid}/answers", {"content": "E2E 的回答内容。"}, T2)
check("回答", s == 200 and r["code"] == 0, r)
aid = r["data"]["id"]
s, r = call("POST", f"/questions/{qid}/accept", {"answer_id": aid}, T1)
check("采纳（含确认语义由前端保证）", s == 200 and r["code"] == 0, r)
s, r = call("POST", f"/questions/{qid}/answers", {"content": "已解决的问题不能再回答"}, T3)
check("已解决后回答 → 422", s == 422 and r["code"] == 42200, (s, r))
s, r = call("POST", f"/questions/{qid}/comments", {"content": "已解决的问题仍可评论"}, T3)
check("已解决后评论仍允许", s == 200 and r["code"] == 0, (s, r))

# ---- 投票 ----
s, r = call("POST", "/votes", {"target_type": "answer", "target_id": aid, "value": 1}, T1)
check("赞同回答", s == 200 and r["data"]["upvote_count"] == 1, r)
s, r = call("POST", "/votes", {"target_type": "answer", "target_id": aid, "value": 1}, T1)
check("同向点击撤销", s == 200 and r["data"]["value"] == 0, r)
s, r = call("POST", "/votes", {"target_type": "answer", "target_id": aid, "value": 1}, T1)
check("重新赞同", s == 200 and r["data"]["upvote_count"] == 1, r)

# ---- 楼中楼评论 ----
s, r = call("POST", f"/questions/{qid}/comments", {"content": "根评论"}, T2)
root_cid = r["data"]["id"]
s, r = call("POST", f"/questions/{qid}/comments", {"content": "楼中楼回复", "parent_id": root_cid}, T1)
check("楼中楼回复", s == 200 and r["code"] == 0 and r["data"]["reply_to_name"] == U2, r)
s, r = call("GET", f"/questions/{qid}/comments")
node = next((n for n in r["data"]["list"] if n["id"] == root_cid), None)
check("树形结构返回回复", node is not None and len(node["replies"]) == 1, r)

# ---- 收藏 ----
s, r = call("POST", f"/questions/{qid}/favorite", None, T2)
check("收藏问题", s == 200 and r["data"]["favorited"] is True, r)
s, r = call("GET", "/me/favorites", token=T2)
fav = [f for f in r["data"]["list"] if f["question_id"] == qid]
check("我的收藏列表", len(fav) == 1, r)
s, r = call("POST", f"/questions/{qid}/favorite", None, T2)
check("取消收藏", s == 200 and r["data"]["favorited"] is False, r)

# ---- 个人主页 ----
s, r = call("GET", f"/users/{U1}/questions")
check("用户提问列表", s == 200 and r["data"]["total"] >= 1, r)
s, r = call("GET", f"/users/{U2}/answers")
check("用户回答列表", s == 200 and r["data"]["total"] >= 1, r)

# ---- 通知 ----
s, r = call("GET", "/notifications", token=T1)
types1 = [n["type"] for n in r["data"]["list"]]
check("被回答通知", "answer" in types1, types1)
check("被评论通知", "comment_question" in types1, types1)
s, r = call("GET", "/notifications", token=T2)
types2 = [n["type"] for n in r["data"]["list"]]
check("被赞同通知", "vote_up_answer" in types2, types2)
check("被回复通知", "comment_reply" in types2, types2)
s, r = call("POST", "/notifications/read-all", token=T1)
check("全部已读", s == 200 and r["code"] == 0, r)

# ---- 筛选 ----
s, r = call("GET", "/questions?accepted=yes&page_size=5")
check("已采纳筛选", s == 200 and all(q["accepted_answer_id"] for q in r["data"]["list"]), r)
s, r = call("GET", "/questions?time=day&sort=hot&page_size=5")
check("时间+热门组合", s == 200 and r["code"] == 0, r)
s, r = call("GET", "/search?q=E2E")
check("全文检索", s == 200 and r["data"]["total"] >= 1, r)

# ---- Token 版本化 ----
s, r = call("PUT", "/users/me/password", {"old_password": "secret123", "new_password": "newpass_e2e"}, T3)
check("修改密码返回新 token", s == 200 and r["data"]["token"], r)
new_token = r["data"]["token"]
s, r = call("GET", "/users/me", token=T3)
check("旧 token 立即失效 → 401", s == 401 and r["code"] == 40100, (s, r))
s, r = call("GET", "/users/me", token=new_token)
check("新 token 可用", s == 200 and r["code"] == 0, (s, r))
s, r = call("POST", "/users/me/refresh", token=new_token)
check("refresh 换发新 token", s == 200 and r["data"]["token"], r)
s, r = call("POST", "/auth/login", {"account": U3, "password": "newpass_e2e"})
check("新密码登录", s == 200 and r["code"] == 0, r)

# ---- metrics ----
import urllib.request as _ur
req = _ur.Request(os.environ.get("BASE", "http://127.0.0.1:8080") + "/metrics")
with _ur.urlopen(req) as resp:
    body = resp.read().decode()
check("metrics 端点", resp.status == 200 and "goask_requests_total" in body, resp.status)

# ---- 权限 ----
s, r = call("PUT", f"/questions/{qid}", {"title": "越权编辑的标题够长了吧哈哈", "content": "x"}, T2)
check("非作者编辑 → 403", s == 403 and r["code"] == 40300, (s, r))
s, r = call("POST", f"/questions/{qid}/close", None, T2)
check("非作者关闭 → 403", s == 403 and r["code"] == 40300, (s, r))

print(f"\n=== 通过 {passed} / 失败 {failed} ===")
exit(1 if failed else 0)
