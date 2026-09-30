# Skills örnekleri

Bu dizin Agent Skills özelliğine ait örnekleri içerir.

## Dizin yapısı

```
skills/
├── README.md              # Bu dosya
└── pdf-processing/        # PDF işleme becerisi örneği
    ├── SKILL.md           # Ana dosya (Level 2)
    ├── FORMS.md           # Ek belge (Level 3)
    └── scripts/           # Çalıştırılabilir betikler
        ├── analyze_form.py
        └── extract_text.py
```

## Hızlı başlangıç

### Demo'yu çalıştırma

```bash
go run ./cmd/skills-demo/main.go
```

### Yeni Skill oluşturma

1. Bu dizinde yeni bir klasör oluşturun:

```bash
mkdir my-new-skill
```

2. `SKILL.md` oluşturun:

```markdown
---
name: my-new-skill
description: Description of what this skill does and when to use it.
---

# My New Skill

Instructions for the agent...
```

3. Betik ekleyin (isteğe bağlı):

```bash
mkdir my-new-skill/scripts
# Betiğinizi ekleyin
```

## Ayrıntılı belgeler

Tüm belgeler için bkz.: [Agent Skills belgeleri](../../website-docs/03-features/22-skills-sandbox.md)

## Örnek: pdf-processing

Bu, şunları gösteren eksiksiz bir örnek beceridir:

- **SKILL.md**: YAML frontmatter içeren ana dosya
- **FORMS.md**: Ek başvuru belgesi
- **scripts/**: Sandbox içinde çalıştırılabilen Python betikleri

### Beceri açıklaması

```yaml
name: pdf-processing
description: Extract text and tables from PDF files, fill forms, merge documents.
```

### İçerilen betikler

| Betik | İşlev |
|------|------|
| `analyze_form.py` | PDF form alanlarını analiz eder |
| `extract_text.py` | PDF'ten metin çıkarır |

### Kullanım örneği

Agent, kullanıcı isteğine göre otomatik olarak çağırır:

```
Kullanıcı: "Bu PDF formunda hangi alanlar var, analiz et"

Agent: 
  1. pdf-processing becerisiyle eşleştiğini tanır
  2. Beceri içeriğini yüklemek için read_file(path="skill://pdf-processing/SKILL.md") çağırır
  3. analyze_form.py'yi çalıştırmak için shell_exec(skill_name="pdf-processing", command=...) çağırır
  4. Form alanı analiz sonucunu döndürür
```
