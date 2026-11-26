#!/usr/bin/env bash
# Имитация уведомлений CloudPayments (подписывает тело так же, как CloudPayments).
# Нужны curl и openssl. Использование:
#   SECRET=<API Secret> ORDER=<номер заказа> AMOUNT=<сумма> bash scripts/cp_webhook_test.sh
set -e
API=${API:-http://localhost:8080}
: "${SECRET:?задайте SECRET (API Secret из CloudPayments, тот же, что в .env)}"
: "${ORDER:?задайте ORDER (номер заказа, например ORD-261126-ABC123)}"
: "${AMOUNT:?задайте AMOUNT (сумма заказа, например 890.00)}"
TX=${TX:-$RANDOM$RANDOM}

send() {
  local path=$1 body=$2
  local sig; sig=$(printf '%s' "$body" | openssl dgst -sha256 -hmac "$SECRET" -binary | base64)
  echo -n "$path -> "
  curl -s -X POST "$API/api/payments/cloudpayments/$path" \
    -H "Content-Type: application/x-www-form-urlencoded" -H "X-Content-HMAC: $sig" --data-binary "$body"
  echo
}

BODY="TransactionId=$TX&Amount=$AMOUNT&Currency=RUB&InvoiceId=$ORDER&Status=Completed&TestMode=1"
send check "$BODY"      # ожидаем {"code":0}
send pay   "$BODY"      # заказ становится paid
send pay   "$BODY"      # повтор: ничего не меняется (идемпотентность)
echo -n "неверная подпись -> "
curl -s -X POST "$API/api/payments/cloudpayments/pay" -H "X-Content-HMAC: AAAA" --data-binary "$BODY"; echo   # ожидаем {"code":13}
