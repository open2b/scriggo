# WIP

## questo documento è attualmente WIP; contiene errori e diverse sezioni devono essere cambiate/riscritte.

# Template in Scriggo

## Introduzione

TODO: scrivere introduzione

Gli esempi verranno mostrati in questo modo:

```
Sorgente del template
```

```
Risultato mostrato dal template
```

## Mostrare testo e HTML

Tutto ciò che viene scritto nel sorgente del template, ad eccezione delle istruzioni e dei commenti, viene mostrato così com'è.

```
Questo testo verrà mostrato così com'è dal template
```

```
Questo testo verrà mostrato così com'è dal template
```

Lo stesso vale per il codice HTML:

```
Testo <b>grassetto</b>
```

```
Testo <b>grassetto</b>
```

Di per sè utilizzare un sistema di template che riscriva il testo così com'è non ha alcuna utilità. Vengono quindi messe a disposizione diverse istruzioni per poter controllare cosa deve essere rappresentato e come deve essere rappresentato.

Si pensi, ad esempio, ad una sezione di una pagina HTML che deve essere mostrata solo in determinate condizioni; oppure si pensi ad calendario che deve essere mostrato in maniera differente in base al giorno.

## Mostrare valori con {{ .. }}

Per mostrare un valore si utilizza `{{ .. }}`; ciò che si trova tra i `{{` e `}}` viene mostrato nel template.

```go
{{ "Hello, World!"}}
```

```
Hello, World!
```

Le istruzioni `{{ .. }}` possono essere inserite intercalate tra del testo:

```go
{{ "Hello" }}, {{ "World!"}}
```

```
Hello, World!
```

All'interno di `{{ .. }}` possono essere inserite delle _espressioni_.

## Espressioni

Le espressioni sono ciò che permette di indicare un valore all'interno di un template.

Alcuni esempi di espressioni sono:

- numeri: `1`, `42`,`-432.11`
- stringhe: `"hello"`
- operazioni matematiche: `3 + 4`, `(5 + 4) * -2.78`

Ad ogni espressione è necessariamente associato un _tipo_. Il tipo definisce il contesto ed il significato di un'espressione. 

### Espressioni numeriche

Le espressioni numeriche sono tutte le espressioni che rappresentano un valore numerico, sia esso un numero naturale, intero, razionale etc..

#### Tipo `int`

Il tipo `int` permette di rappresentare i numeri interi, come ad esempio:

- `42`
- `-100`
- `0`

È possibile comporre delle espressioni intere mediante gli operatori matematici `+`, `-`, `*`, `/` e `%`.

Prendiamo ad esempio `40 + 2`; sia `40` che `2` sono entrambe espressioni intere di tipo `int`. L'operatore `+` effettua la somma tra i due numeri interi, restituendo a sua volta un valore di tipo `int`.

Possiamo quindi usare il template per svolgere calcoli matematici:

```
In un anno ci sono {{ 365 * 24 * 3600 }} secondi.
```
```
In un anno ci sono 31536000 secondi.
```

#### Tipo `float64`

Il tipo `float64` permette di rappresentare numeri in virgola mobile.

```
{{ 56 + 321.43 * 43 }}
```
```
13877.49
```

#### Ulteriori tipi numerici

Nel sistema di template in Scriggo sono disponibili ulteriori tipi numerici. Per approfondire vedi 

<!-- Per confrontare i numeri sono disponibili gli operatori <, <=, ==, >= e >. -->

### Stringhe

Il tipo `string` permette di rappresentare stringhe, ovvero sequenze di caratteri Unicode.

Una stringa viene scritta all'interno dei doppi apici `"` oppure all'interno di due caratteri accento grave.

L'operatore `+`, applicato alle stringhe, permette di concatenare due stringhe per ottenere una nuova stringa.

```go
{{ "Scrig" + "go" }}
```
```
Scriggo
```

Questo è un esempio di come il tipo interviene sul modo in cui le espressioni vengono interpretate: se sui tipi `int` l'operatore effettua la somma tra i due numeri, sui tipi `string` effettua la concatenazione.

```
Tipi int:    {{ 4 + 4 }}
Tipi string: {{ "4" + "4" }}
```
```
Tipi int:    8
Tipi string: 44
```

Ogni volta che viene utilizzato un operatore il tipo a sinistra deve avere lo stesso tipo a destra. Applicare l'operatore `+` ad un tipo `int` e ad una stringa porta ad un errore.


```
{{ 10 + "7" }}
```
```
ERRORE: impossibile effettuare l'operazione int + string
```

### Slice

Le espressioni di tipo `slice` permettono di rappresentare una _sequenza_ di espressioni.

Uno slice viene costruito mediante

```go
[]tipo{elemento1, elemento2, ... elementoN}
```

Ad esempio uno slice di `int` contiene una sequenza di espressioni `int`:

```go
[]int{3, 432, -4, 843, 0, 48, 44}
```

Uno slice, così come le altre espressioni, può essere specificato anche su più righe:

```go
[]int{
    3,
    432,
    -4,
    843,
    0,
    48,
    44,
}
```

I tipi `slice` verrano ripresi più avanti.

### Map

Le espressioni di tipo map permettono di rappresentare una struttura di tipo chiave-valore.

Un map viene definito con la sintassi:
```go
map[tipoChiave]tipoValore{chiave1: valore1, chiave2: valore2 ... }
```

Supponiamo di voler rappresentare una struttura che associ, ad ogni prodotto, il prezzo unitario. A questo scopo utilizzato un `map` da `string` (nome del prodotto) a `float64` (prezzo unitario, in euro):

```go
map[string]float64{

}
```

I tipi `map` verrano ripresi più avanti.


## Variabili

Una variabile è un contenitore per un valore: possiamo quindi assegnare ad una variabile un certo valore per poi leggerlo nuovamente in seguito.

Per _dichiarare_ una nuova variabile si utilizza la sintassi

```
{% variabile := espressione %}
```

Una variabile, una volta definita, può essere usata come espressione.

Vediamo subito un esempio per chiarire l'utilizzo delle variabili.

<!-- TODO: variabili builtin -->
<!-- TODO: assegnamento multiplo -->



---

# Struttura di riferimento

    • Introduzione
    • Show {{ }}
    • Espressioni
        ◦ Cos’è un espressione
        ◦ Tipo di un’espressione
        ◦ Come definire un espressione
            ▪ Numeri, stringhe, booleani
            ▪ Slice, map
        ◦ Espressioni unarie
        ◦ Espressioni binarie
    • Variabili
        ◦ Definizione di variabili tramite :=
        ◦ Definizione di variabili tramite var
            ▪ Definizione di variabili con tipo
        ◦ Assegnamento a variabili tramite =
        ◦ Usare le variabili come espressioni
    • Statement
        ◦ {% if %}
            ▪ {% else %}
        ◦ {% for %}
            ▪ {% for .. in .. %}
            ▪ {% for .. range .. %}
            ▪ {% for ; ; ; %}
            ▪ {% break %}
            ▪ {% continue %}
        ◦ {% switch %}
            ▪ {% break %}
            ▪ {% fallthrough %}
    • Definizione e chiamata di macro
        ◦ Macro senza argomenti
        ◦ Macro con argomenti
    • Struttura
        ◦ {% include %}
        ◦ {% import %}
        ◦ {% extends %}
    • Builtin del template e di Go
        ◦ len
        ◦ builtin1
        ◦ builtin2
        ◦ builtin3
    • Avanzate
        ◦ Chiamata di funzioni dal template
            ▪ Le chiamate di funzione sono espressioni (con esempi)
            ▪ Funzioni con valori multipli di ritorno
            ▪ Definizione di funzioni nel template
        ◦ Statement “panic”
        ◦ Controllo degli errori tramite “if err != nil ...”
            ▪ Tipo “error”
        ◦ Statement “defer”
            ▪ Builtin “recover”
        ◦ Differenze tra HTML e string nel rendering
        ◦ Assegnamento tramite ++, –, +=, -= …
        ◦ Accenni alle costanti
        ◦ Commenti {# #}
        ◦ Array
        ◦ Concorrenza
            ▪ Statement {% go %}
            ▪ Statement {% select %}

