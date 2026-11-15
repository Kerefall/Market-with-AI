package ai

import (
	"fmt"
	"strings"
)

// ВАЖНО: промпты вынесены в одно место. Если в документе с промптами тексты другие,
// достаточно заменить константы ниже: остальной код менять не нужно.

const DescriptionSystem = `Ты — копирайтер интернет-магазина. Пиши по-русски.
Правила: не выдумывай характеристики, которых нет во входных данных; без markdown и эмодзи;
объём 80–150 слов; первый абзац — польза для покупателя, второй — ключевые характеристики;
естественно используй ключевые слова для SEO, не повторяя их подряд.`

const PriceSystem = `Ты — помощник селлера по ценообразованию. Пиши по-русски, коротко (3–5 предложений).
Тебе даны реальные цены похожих товаров на площадке. Не придумывай других цифр.
Объясни, как рекомендуемая цена соотносится с рынком, и дай один практический совет.`

func descriptionUser(name, category, tone string, attrs []Attribute, keywords []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Товар: %s\n", name)
	if category != "" {
		fmt.Fprintf(&b, "Категория: %s\n", category)
	}
	if len(attrs) > 0 {
		b.WriteString("Характеристики:\n")
		for _, a := range attrs {
			fmt.Fprintf(&b, "- %s: %s\n", a.Name, a.Value)
		}
	}
	if len(keywords) > 0 {
		fmt.Fprintf(&b, "Ключевые слова для SEO: %s\n", strings.Join(keywords, ", "))
	}
	fmt.Fprintf(&b, "Тон текста: %s\n", tone)
	b.WriteString("Напиши описание товара для карточки.")
	return b.String()
}

func priceUser(name, category, description string, st priceStats) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Товар: %s\n", name)
	if category != "" {
		fmt.Fprintf(&b, "Категория: %s\n", category)
	}
	if description != "" {
		fmt.Fprintf(&b, "Описание: %s\n", description)
	}
	fmt.Fprintf(&b, "Похожих товаров на площадке: %d. Цены: минимум %.2f, средняя %.2f, максимум %.2f.\n",
		st.Count, st.Min, st.Avg, st.Max)
	fmt.Fprintf(&b, "Рекомендуемая цена: %.2f.\nОбоснуй рекомендацию.", st.Suggested)
	return b.String()
}

// Резервные тексты демо-режима (когда ИИ-провайдер не подключён).
func fallbackDescription(name, category string, attrs []Attribute) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s — практичный выбор", name)
	if category != "" {
		fmt.Fprintf(&b, " в категории «%s»", category)
	}
	b.WriteString(". Товар подойдёт для ежедневного использования и станет хорошим подарком.")
	if len(attrs) > 0 {
		b.WriteString(" Основные характеристики: ")
		parts := make([]string, 0, len(attrs))
		for _, a := range attrs {
			parts = append(parts, fmt.Sprintf("%s — %s", a.Name, a.Value))
		}
		b.WriteString(strings.Join(parts, "; "))
		b.WriteString(".")
	}
	b.WriteString(" (Демо-режим ИИ: подключите LLM_API_URL для настоящей генерации.)")
	return b.String()
}

func fallbackPrice(st priceStats) string {
	return fmt.Sprintf("На площадке найдено похожих товаров: %d. Цены от %.2f до %.2f, средняя %.2f. "+
		"Рекомендуем ориентироваться на %.2f. (Демо-режим ИИ.)", st.Count, st.Min, st.Max, st.Avg, st.Suggested)
}
