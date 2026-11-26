#!/usr/bin/env bash
# Сквозная проверка API. Нужны curl и jq. Запуск: bash scripts/smoke.sh
set -e
API=${API:-http://localhost:8080}
j() { curl -s -H 'Content-Type: application/json' "$@"; }

echo "== health";            j $API/health
echo; echo "== вход селлера"
TOKEN=$(j -X POST $API/api/auth/login -d '{"email":"garden@shop.test","password":"seller12345"}' | jq -r .token)
AUTH="Authorization: Bearer $TOKEN"
echo "token: ${TOKEN:0:20}..."

echo "== мой магазин";       j -H "$AUTH" $API/api/shop | jq -c .
echo "== роль: seller → admin (ожидаем 403)"
curl -s -o /dev/null -w "%{http_code}\n" -H "$AUTH" $API/api/admin/users

echo "== каталог (без аккаунта)"
j "$API/api/catalog/products?limit=3" | jq -c '{total, names: [.items[].name]}'

echo "== создать товар"
PID=$(j -X POST -H "$AUTH" $API/api/seller/products \
  -d '{"name":"Тестовая лопата","price":1200,"stock":5,"category_id":5,"images":["https://placehold.co/600"],"attributes":[{"name":"Материал","value":"сталь"}]}' | jq -r .id)
echo "product id: $PID"

echo "== ИИ: описание и цена"
j -X POST -H "$AUTH" $API/api/ai/description -d "{\"product_id\":$PID}" | jq -c .
j -X POST -H "$AUTH" $API/api/ai/price -d "{\"product_id\":$PID}" | jq -c .

echo "== заказ покупателя (без аккаунта)"
ORDER=$(j -X POST $API/api/orders -d '{
  "customer_name":"Тест Покупатель","customer_phone":"+79990000000","customer_email":"buyer@example.com",
  "delivery_address":"Екатеринбург, ул. Тестовая, 1",
  "items":[{"product_id":1,"quantity":1}]}')
echo "$ORDER" | jq -c '.orders[] | {id, number, total_amount, status}'
NUM=$(echo "$ORDER" | jq -r '.orders[0].number'); OID=$(echo "$ORDER" | jq -r '.orders[0].id')

echo "== тестовая оплата"
j -X POST $API/api/orders/$NUM/pay -d '{"email":"buyer@example.com"}' | jq -c .

echo "== селлер: new → assembling"
j -X PATCH -H "$AUTH" $API/api/seller/orders/$OID/status -d '{"status":"assembling"}' | jq -c '{number,status,payment_status}'

echo "== аналитика"
j -H "$AUTH" "$API/api/seller/analytics?days=30" | jq -c .

echo "== админ: модерация"
ATOKEN=$(j -X POST $API/api/auth/login -d '{"email":"admin@shop.test","password":"admin12345"}' | jq -r .token)
j -H "Authorization: Bearer $ATOKEN" $API/api/admin/moderation/products | jq -c '[.[] | {id,name,status}]'
echo "OK"
