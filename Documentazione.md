<!-- TODO: uniformare {% end .. %} -->

<!-- <style>
pre.example hr {
    border-width: 3px;
    margin-top: 25px;
}
pre.example {
    border-style: solid;
    border-width: 2px;
    border-radius: 2px;
    border-color: #A8E6FF;
    background-color: #ECFAFF;
}
</style> -->

# Template in Scriggo

  * [Introduzione](#introduzione)
  * [Mostrare valori nel template](#mostrare-valori-nel-template)
  * [Espressioni](#espressioni)
    + [Espressioni numeriche](#espressioni-numeriche)
      - [Tipo int](#tipo-int)
      - [Tipo float64](#tipo-float64)
    + [Stringhe](#stringhe)
    + [Booleani](#booleani)
    + [Slice](#slice)
    + [Map](#map)
  * [Variabili](#variabili)
    + [Dichiarazione di variabili tramite var](#dichiarazione-di-variabili-tramite-var)
    + [Assegnamento di un nuovo valore](#assegnamento-di-un-nuovo-valore)
  * [Controllo di flusso](#controllo-di-flusso)
    + [Istruzione if](#istruzione-if)
      - [If else](#if-else)
    + [Istruzione for](#istruzione-for)
      - [break in un for](#break-in-un-for)
      - [continue in un for](#continue-in-un-for)
      - [for .. range](#for--range)
      - [for ; ; ;](#for------)
    + [Istruzione switch](#istruzione-switch)
      - [break in uno switch](#break-in-uno-switch)
      - [fallthrough in uno switch](#fallthrough-in-uno-switch)
  * [Macro](#macro)
    + [Macro con argomenti](#macro-con-argomenti)
  * [Template su file multipli](#template-su-file-multipli)
    + [Istruzione include](#istruzione-include)
    + [Istruzione import](#istruzione-import)
    + [Istruzione extends](#istruzione-extends)
  * [Builtin](#builtin)
    + [len](#len)
  * [Avanzate](#avanzate)
    + [Chiamata di funzioni dal template](#chiamata-di-funzioni-dal-template)
    + [Commenti](#commenti)
    + [Builtin panic](#builtin-panic)
    + [Controllo degli errori nel template in Scriggo](#controllo-degli-errori-nel-template-in-scriggo)
    + [Scorciatoie per l'assegnamento](#scorciatoie-per-l-assegnamento)
    + [Accenni alle costanti](#accenni-alle-costanti)
    + [Array](#array)
    + [Controllo sul rendering](#controllo-sul-rendering)
      - [Differenza tra HTML e string](#differenza-tra-html-e-string)
    + [Concorrenza](#concorrenza)
      - [Istruzione go](#istruzione-go)
      - [Istruzione select](#istruzione-select)
    + [Istruzione defer](#istruzione-defer)
      - [Builtin recover](#builtin-recover)

**TODO: questo documento è attualmente WIP, pertanto contiene errori, imprecisioni, omissioni e sezioni da rimuovere**

## Introduzione

**TODO: aggiungere un introduzione.**

Gli esempi verranno mostrati in questo modo:

<pre class="example">
Sorgente del template
<hr>Risultato mostrato dal template
</pre>

> NOTA: per una questione di leggibilità, alcuni esempi potrebbero contenere degli _a capo_ o degli spazi che non verrebbero in realtà renderizzati, o viceversa, vengono nascoste delle spaziature che invece sono presenti nel template mostrato da Scriggo. Per approfondire la questione degli spazi si veda la sezione [Avanzato - Rendering del testo]()

<!-- ## Mostrare testo e HTML

Tutto ciò che viene scritto nel sorgente del template, ad eccezione delle istruzioni e dei commenti, viene mostrato così com'è.

<pre class="example">
Questo testo verrà mostrato così com'è dal template
<hr>Questo testo verrà mostrato così com'è dal template
</pre>

Lo stesso vale per il codice HTML:

<pre class="example">
Testo &lt;b&gt;grassetto&lt;/b&gt;
<hr>Testo &lt;b&gt;grassetto&lt;/b&gt;
</pre>

Di per sè utilizzare un sistema di template che riscriva il testo così com'è non ha alcuna utilità. Vengono quindi messe a disposizione diverse istruzioni per poter controllare cosa deve essere rappresentato e come deve essere rappresentato.

Si pensi, ad esempio, ad una sezione di una pagina HTML che deve essere mostrata solo in determinate condizioni; oppure si pensi ad calendario che deve essere mostrato in maniera differente in base al giorno. -->

## Mostrare valori nel template

Per mostrare un valore si utilizza `{{ .. }}`; ciò che si trova tra i `{{` e `}}` viene mostrato nel template.

<pre class="example">
{{ "Hello" }}, {{ "World!" }}
<hr>Hello, World!
</pre>

## Espressioni

Ad ogni espressione è necessariamente associato un _tipo_. Il tipo definisce il contesto ed il significato di un'espressione. 

### Espressioni numeriche

Le espressioni numeriche sono tutte le espressioni che rappresentano un valore numerico, sia esso un numero naturale, intero, razionale etc..

#### Tipo int

Il tipo `int` rappresenta numeri interi, sui quali è possibile svolgere operazioni matematiche.

<pre class="example">
In un anno ci sono {{ 365 * 24 * 3600 }} secondi.
<hr>In un anno ci sono 31536000 secondi.
</pre>

#### Tipo float64

Il tipo `float64` permette di rappresentare numeri con parte decimale.

<pre class="example">
{{ 56 + 321.43 * 43 }}
<hr>13877.49
</pre>

> Nel sistema di template in Scriggo sono disponibili ulteriori tipi numerici, tutti quelli disponibili in Go. Per approfondire vedi l'[elenco di builtin in Go](https://golang.org/pkg/builtin/).

<!-- Per confrontare i numeri sono disponibili gli operatori <, <=, ==, >= e >. -->

### Stringhe

Il tipo `string` rappresenta delle stringhe.

Una stringa viene scritta all'interno dei doppi apici `"` oppure all'interno di due caratteri accento grave.

L'operatore `+`, applicato alle stringhe, permette di concatenare due stringhe per ottenere una nuova stringa.

<pre class="example">
{{ "Scrig" + "go" }}
<hr>Scriggo
</pre>

Questo è un esempio di come il tipo interviene sul modo in cui le espressioni vengono interpretate: se sui tipi `int` l'operatore effettua la somma tra i due numeri, sui tipi `string` effettua la concatenazione.

<pre class="example">
Tipi int:    {{ 4 + 4 }}
Tipi string: {{ "4" + "4" }}
<hr>Tipi int:    8
Tipi string: 44
</pre>

Ogni volta che viene utilizzato un operatore il tipo a sinistra deve avere lo stesso tipo a destra. Applicare l'operatore `+` ad un tipo `int` e ad una stringa porta ad un errore.


<pre class="example">
{{ 10 + "7" }}
<hr>ERRORE: impossibile effettuare l'operazione int + string
</pre>

### Booleani

Il tipo `bool` rappresenta un valore booleani (`true` o `false`). Gli operatori di confronto restituiscono un valore di tipo `bool`.

<pre class="example">
{{ 5 == 2 + 3 }}
{{ "hello" == "ciao" }}
{{ 43.432 > 0 }}
<hr>true
false
true
</pre>

### Slice

Le espressioni di tipo `slice` permettono di memorizzare una lista di valori, tutti dello stesso tipo.

```
[]int{-2, 4, 0}
```

Uno slice, così come le altre espressioni, può essere specificato anche su più righe:

```
[]int{
   -2,
   4,
   0,
}
```

### Map

I **map** permettono di memorizzare delle coppie chiave-valore.

Supponiamo di voler rappresentare una struttura che associ, ad ogni prodotto, il prezzo unitario. A questo scopo utilizzato un `map` da `string` (nome del prodotto) a `float64` (prezzo unitario, in euro):

```
map[string]float64{
    "Mele": 0.80,
    "Pere": 0.75,
    "Cocomero": 8.99,
}
```

> NOTA: i map, a differenza degli slice, non hanno un criterio di ordinamento. Questo significa che _iterando_ sulle chiavi di un map il criterio di ordinamento non è specificato.

## Variabili

Una variabile è un contenitore per un valore: possiamo quindi assegnare ad una variabile un certo valore per poi leggerlo nuovamente in seguito.

Per **dichiarare una nuova variabile** si utilizza la sintassi

```
{% variabile := espressione %}
```

Una variabile, una volta definita, può essere usata come espressione.

Vediamo subito un esempio per chiarire l'utilizzo delle variabili.

<!-- TODO: variabili builtin -->
<!-- TODO: assegnamento multiplo -->

<pre class="example">
{% name := "John" %}
Il nome è {{ name }}.
<hr>Il nome è John.
</pre>

La prima riga `{% name := "John" %}` assegna un valore di tipo `string` con valore `"John"` ad una nuova variabile chiamata `name`. Si ha quindi che la variabile `name` è un espressione di tipo `string` che può essere usata come una qualsiasi altra stringa:

<pre class="example">
{% name := "John" %}
{{ name }}, {{ name + name + name }}
<hr>John, JohnJohnJohn
</pre>

Ovviamente più variabili possono essere definite:

<pre class="example">
{% name := "John" %}
{% surname := "Smith" %}
Hello {{ name }} {{ surname }}!
<hr>Hello John Smith!
</pre>

### Dichiarazione di variabili tramite var

Le variabili possono anche essere dichiarate mediante una sintassi alternativa

<pre>
var <em>variabile</em> = <em>expressione</em>
</pre>

Si noti che la parola chiave `var` precede il nome della variabile, e che al posto dell'operatore `:=` è stato usato l'operatore `=`.

Riscriviamo quindi l'esempio di prima:

<pre class="example">
{% var name = "John" %}
Il nome è {{ name }}.
<hr>Il nome è John.
</pre>

La dichiarazione di variabili tramite `var`, rispetto a `:=`, ha la caratteristica di supportare anche il **tipo nella dichiarazione di variabile**.

<pre class="example">
{% var name string = "John" %}
Il nome è {{ name }}.
<hr>Il nome è John.
</pre>

Questa nuova sintassi, seppure più prolissa, permette di avere un maggior controllo di correttezza dei tipi. In particolare `name`, nell'esempio precedente, aveva tipo `string` in quanto le veniva assegnata un'espressione di tipo `string`. In questo caso il tipo viene indicato esplicitamente durante la dichiarazione.

Se proviamo quindi a dichiarazione una variabile `string` ed ad assegnarle un espressione di tipo `int` otteniamo un errore di tipo:

<pre class="example">
{% var name string = 30 %}
Il nome è {{ name }}.
<hr>ERRORE: impossibile assegnare 30 (tipo int) alla variabile name (tipo string)
</pre>

### Assegnamento di un nuovo valore

L'assegnamento di un valore ad una variabile viene effettuato tramite l'operatore `=`.

<pre class="example">
{% name := "John" %}
Il nome è {{ name }}.
{% name = "Paul" %}
Il nome è {{ name }}.
<hr>Il nome è John.
Il nome è Paul.
</pre>

### Assegnamento multiplo

Più variabili possono essere assegnate (o dichiarate) in una sola istruzione:

<pre>
{% a, b, c := 10, 20, 30 %}
{% <b>var</b> d, e, f = 40, 50, 60 %}
</pre>


## Controllo di flusso

Il template di Scriggo mette a disposizione tre istruzioni principali per controllare il flusso di rendering del template:

- L'struzione **if** mostra una parte del template solo se una determinata condizione è verificata
- L'istruzione **for** mostra ripetutamente la stessa parte di template
- L'istruzione **switch** mostra una differente parte del template in base ad una determinata condizione

> NOTA: Sono presenti anche altre istruzioni più avanzate per il controllo del flusso di programma, che verranno discusse in seguito.

### Istruzione if

L'istruzione `{% if .. %}` permette di mostrare una parte del template in base ad una determinata condizione.

<pre class="example">
{% <b>if</b> 2 + 2 == 4 %}
    La matematica funziona!
{% <b>end if</b> %}
<hr>    La matematica funziona!
</pre>

Nella condizione dell'**if** si possono ovviamente usare anche le variabili:

<pre class="example">
{% name := "George" %}
{% <b>if</b> name == "John" %}
    Ciao, Lennon!
{% <b>end if</b> %}
<hr>(non è viene stato mostrato nulla)
</pre>


#### If else

Supponiamo di avere una variabile `isLoggedIn` di tipo `bool` che contiene il valore `true` se l'utente ha effettuato il login, oppure `false` se non l'ha effettuato.
Possiamo quindi visualizzare un messaggio di benvenuto diverso in base a questa condizione (supponiamo, in questo esempio, che l'utente non abbia effettuato il login, e che pertanto la variabile `isLoggedIn` abbia valore `false`):

<pre class='example'>
{% <b>if</b> isLoggedIn %}
    Benvenuto, utente registato!
{% <b>else</b> %}
    Non hai ancora effettuato il login, cosa aspetti?
{% <b>end if</b> %}
<hr>Non hai ancora effettuato il login, cosa aspetti?
</pre>

#### If else if

Nell'istruzione **if** è possibile aggiungere anche **else if**:

<pre class='example'>
{% <b>if</b> 4 == 5 %}
    Sbagliato..
{% <b>else if</b> 4 > 5 %}
    Sbagliato..
{% <b>else if</b> 4 < 5 %}
    Corretto!
{% <b>else</b> %}
    Impossibile arrivare qui, almeno una delle condizioni
    precedenti deve essersi verificata..
{% <b>end if</b> %}
<hr>Corretto!
</pre>


### Istruzione for

L'istruzione `for` permette di mostrare più volte una sezione di template. Questo è particolarmente utile nel caso in cui si abbia uno _slice_ di valori, e per ognuno di essi debba essere mostrata una parte di template.

Supponiamo di avere una variabile `products` di tipo `[]string`, ovvero uno _slice di stringhe_.
Possiamo inserire questa variabile nel contenuto di un **for**, che mostrerà una parte di template ripetutamente, per ogni stringa contenuta in `products`.

<pre class='example'>
{% products := []string{"Frigorifero", "Lavastoviglie", "Forno"} %}
{% <b>for</b> product <b>in</b> products %}
    Acquista {{ product }}!
{% <b>end for</b> %}
<hr>Acquista Frigorifero!
Acquista Lavastoviglie!
Acquista Forno!
</pre>

L'istruzione `for` può contenere al suo interno anche altre istruzioni.
Possiamo riscrivere l'esempio precedente in questo modo:
<pre class='example'>
{% <b>for</b> product <b>in</b> products %}
    Acquista
    {% <b>if</b> product == "Lavastoviglie" %}
        una
    {% <b>else</b> %}
        un
    {% <b>end if</b> %}
    {{ product }}!
{% <b>end for</b> %}
<hr>Acquista un Frigorifero!
Acquista una Lavastoviglie!
Acquista un Forno!
</pre>

> NOTA: l'istruzione **for .. in ..** è l'unico caso di uno statement che non esiste in Go ma che è possibile utilizzare nel template in Scriggo.

#### break in un for

L'istruzione `{% break %}` interrompe il **for** dentro al quale ci si trova.

<pre class='example'>
{% numbers := []int{1, 2, 3, 4, 5, 6 } %}

{% <b>for</b> n <b>in</b> numbers %}

    Numero: {{ n }}

    {% <b>if</b> n == 3 %}
        {% <b>break</b> %}
    {% <b>end if</b> }

{% <b>end for</b> %}

Fine!
<hr>
Numero: 1
Numero: 2
Numero: 3
Fine!
</pre>


#### continue in un for

L'istruzione `{% continue %}` permette di proseguire con la prossima iterazione del **for** interrompendo quella attuale.

<pre class='example'>
{% <b>for</b> v <b>in</b> []int{1, 2, 3} %}
    {{ v }}
    {% <b>if</b> v % 2 == 0 %}
        {% continue %}
    {% <b>end if</b> %}
    è un numero dispari
{% end for %}
<hr>
1 è un numero dispari
2
3 è un numero dispari
</pre>


#### for .. range

Esiste una variante del ciclo for che permette di memorizzare anche l'indice al quale si trova un dato valore all'interno dello slice sul quale il `for` andrà ad iterare.

<pre class='example'>
{% <b>for</b> i, product := <b>range</b> products %}
    {{ i }}. {{ product }} 
{% <b>end for</b> %}
<hr>    0. Frigorifero
    1. Lavastoviglie
    2. Forno
</pre>

In questo caso alla variabile `i` viene assegnato l'_indice_ al quale si trova ogni prodotto all'interno della variabile `products`. La variabile `i` sarà implicitamente dichiarata di tipo `int`.

Dal momento che gli **indici partono da 0**, possiamo riscrivere l'esempio precedente sfruttando ciò che abbiamo visto sui numeri interi, ovvero che possono essere utilizzati all'interno di espressioni più complesse mediante operatori matematici. 

<pre class='example'>
{% <b>for</b> i, product := <b>range</b> products %}
    {{ i + 1 }}. {{ product }} 
{% <b>end for</b> %}
<hr>    1. Frigorifero
    2. Lavastoviglie
    3. Forno
</pre>

#### for classico

Il template in Scriggo supporta anche il _classico_ ciclo for, comunemente diffuso in diversi linguaggi di programmazione.

<pre class='example'>
{% <b>for</b> i := 0; i < 3; i++ %}
    Ciclo {{ i }}
{% <b>end for</b> %}
<hr>
Ciclo 0
Ciclo 1
Ciclo 2
</pre>

Anche in questo tipo di **for** sono disponibili le istruzioni **continue** e **break**.

### Istruzione switch

L'istruzione `switch` seleziona una parte di template da visualizzare in base al valore di un'espressione.

Vediamo un esempio nel quale deve essere mostrato un messaggio diverso in base al giorno della settimana. Supponiamo quindi di avere una variabile `department` che conterrà il nome del reparto di un dato prodotto.

<pre class='example'>
{% department := "Elettrodomestici" }

{% <b>switch</b> department %}

    {% <b>case</b> "Abiti" %}
        Foto di un giacchetto

    {% <b>case</b> "Elettrodomestici" %}
        Foto di un frullatore

    {% <b>default</b> %}
        Nessuna foto disponibile per questo reparto

{% <b>end switch</b> %}
<hr>Foto di un frullatore
</pre>

#### break in uno switch

#### fallthrough in uno switch

## Macro

Le **macro** permettono di definire un "blocco" di template che può essere mostrato più volte altrove, nella posizione desiderata.

<pre class='example'>
{% <b>macro</b> Banner %}
    Acquista oggi un nuovo Frigorifero!
    Foto di un frigorifero
{% <b>end</b> macro %}

Titolo

{% <b>show</b> Banner %}

Contenuto della pagina 

{% <b>show</b> Banner %}
<hr>Titolo

    Acquista oggi un nuovo Frigorifero!
    Foto di un frigorifero

Contenuto della pagina 

    Acquista oggi un nuovo Frigorifero!
    Foto di un frigorifero
</pre>

La sintassi per mostrare una **macro** è:

<pre>
{% <b>show</b> NomeMacro %}
</pre>

oppure

<pre>
{{ NomeMacro }}
</pre>

Quest'ultima sintassi è la stessa utilizzata per mostrare delle espressioni. Le due sintassi sono tra loro equivalenti.

### Macro con argomenti

Le macro offrono la possibilità di specificare dei parametri in ingresso, che possono quindi variare ogni volta in cui la macro viene chiamata.

<pre class='example'>
{% <b>macro</b> Banner(product string) %}
    Acquista oggi un nuovo {{ prodotto }}
    Foto di un {{ product }}
{% <b>end</b> macro %}

Titolo

{% <b>show</b> Banner("Frigorifero") %}

Contenuto della pagina 

{% <b>show</b> Banner("Forno") %}
<hr>Titolo

    Acquista oggi un nuovo Frigorifero!
    Foto di un Frigorifero

Contenuto della pagina 

    Acquista oggi un nuovo Forno!
    Foto di un Forno
</pre>


## Template su file multipli

Il template in Scriggo mette a disposizione tre istruzioni per far si che un unico template possa essere diviso su più file.

- istruzione **include**: include un altro file così com'è
- istruzione **import**: importa le _dichiarazioni_ di un altro file
- istruzione **extends**: _estende_ un altro file

Vediamo in dettaglio queste tre istruzioni.

### Istruzione include

L'istruzione **include** permette di includere un file di template così com'è, come se il suo contenuto fosse copiato ed incollato al posto dell'istruzione **include**.

<pre class="example">
Contenuto incluso:
{% include "file.html %}
<hr>
Contenuto incluso:
Contenuto di 'file.html'!
</pre>

### Istruzione import

**TODO**

### Istruzione extends

**TODO**

## Builtin

Nel sistema di template in Scriggo vengono fornite delle dichiarazioni **builtin**, ovvero dichiarazioni che sono implicitamente definite in ogni file del template.

**TODO: aggiungere le builtin**

### len

`len(expr)` ritorna la lunghezza di `expr`.

- Se `expr` è una stringa, viene restituito il numero di byte che la compongono.
- Se `expr` è uno slice, viene restituito il numero di elementi che lo compongono.
- Se `expr` è un map, viene restituito il numero di coppie chiave-valore che lo compongono.

## Argomenti avanzati

### Chiamata di funzioni dal template

Dal template è possibile effettuare chiamate alle funzioni.

```
{% deleteFile("test.txt") %}
```

Nel caso particolare in cui una funzione abbia un solo valore di ritorno, la chiamata a tale funzione può essere usata come _espressione_.

<pre class='example'>
Quest'anno è il {{ year() }},
il prossimo sarà il {{ year() + 1 }}.
<hr>Quest'anno è il 2047,
il prossimo sarà il 2048.
</pre>

<!-- ▪ Le chiamate di funzione sono espressioni (con esempi)
▪ Funzioni con valori multipli di ritorno
▪ Definizione di funzioni nel template -->

### Commenti

Nel template è possibile specificare sezioni di codice che non verranno renderizzati mediante la sintassi

<pre>
{# commento #}
</pre>

I commenti possono essere scritti su più righe:

<pre>
{#
    prima riga del commento
    seconda riga del commento
#}
</pre>

Vediamo un esempio

<pre class='example'>
Testo da mostrare {# primo commento #}
{# secondo commento #}
Altro testo da mostrare {# terzo 

commento #}
<hr>Testo da mostrare
Altro testo da mostrare
</pre>

**Differenza tra commenti del template di Scriggo e commenti HTML**

Nel caso in cui il template andrà a renderizzare codice HTML, può sorgere la domanda:

> Quali commenti utilizzo? Quelli del template (`{# .. #}`) o quelli dell'HTML (<code>&lt;!-- .. --&gt;</code>)?

La risposta è che dipende dal risultato che si vuole ottenere.

I commenti inseriti nel template non vengono renderizzati, quindi, se ad esempio il template viene renderizzato lato server, i commenti `{# .. #}` non verranno mai inviati al client.

Al contrario, i commenti HTML <code>&lt;!-- .. --&gt;</code> vengono ignorati dal template e vengono trattati come testo, facendo in modo che essi poi compaiano nel sorgente della pagina HTML inviata al client.

### Builtin panic

Nel template è disponibile una builtin speciale chiamata `panic`.
Accetta un solo argomento, e nel momento in cui viene chiamata il processo di rendering del template viene arrestato, mostrando l'argomento passato come messaggio d'errore.

La builtin `panic` viene in genere chiamata in situazioni anomale che impediscono di proseguire con il rendering del template.

<pre class='example'>
panic("Il file non può essere letto")

Testo da mostrare..
<hr>
PANIC: si è verificato un errore: Il file non può essere letto
</pre>

### Controllo degli errori nel template in Scriggo

Il template in Scriggo prevede un controllo degli errori uguale a quello utilizzato in Go.

È quindi previsto che le funzioni che possono andare in errore ritornino quest'ultimo come ultimo parametro.

```
{% content, err := readFile("test.txt") %}
```

In questo caso, se la lettura del file non è andata a buon fine, la variabile `err` conterrà l'errore che si è verificato; questo può essere controllato con un'istruzione **if**, ed in caso in cui l'errore sia effettivamente `!= nil`, è possibile chiamare la builtin `panic` per arrestare il rendering del template.

<pre>
{% content, err := readFile("test.txt") %}
{% <b>if</b> err != nil %}
    {% panic(err) %}
{% <b>end if</b> %}
</pre>

> Per approfondire vedi [Effective Go - Gestione degli errori](https://golang.org/doc/effective_go.html#errors).

### Scorciatoie per l'assegnamento

**TODO: ++, --, += etc..**

### Accenni alle costanti

**TODO**

### Array

**TODO**

### Controllo sul rendering

**TODO**

#### Differenza tra HTML e string

Se ci si trova nel contesto HTML e si renderizza una stringa, il codice HTML contenuto al suo interno verrà <em>sottoposto ad escape</em>. Questo fa si che non sia possibile modificare la struttura della pagina HTML mediante il rendering di stringhe.

Ad esempio:

<pre>
{% s := &quot;&lt;b&gt;bold text&lt;/b&gt;&quot; %}
{{ s }}
</pre>

viene renderizzato nella pagina HTML come

<pre>
&lt;b&gt;bold text&lt;/b&gt;
</pre>

in quanto è stato <em>sottoposto ad escape</em>.

Se questo comportamento non è quello desiderato, ovvero si vuole che il contenuto di una stringa
venga renderizzato _così com'è_ è necessario indicare esplicitamente il tipo `HTML` durante il rendering:

<pre>
{% s := HTML(&quot;&lt;b&gt;bold text&lt;/b&gt;&quot;) %}
{{ s }}
</pre>

in questo modo il browser effettuerà correttamente il rendering del codice HTML in quanto questo non è stato <em>sottoposto ad escape</em>:

<pre>
<b>bold text</b>
</pre>

### Concorrenza

#### Istruzione go

**TODO**

#### Istruzione select

**TODO**

### Istruzione defer

L'istruzione `defer` permette rimandare l'esecuzione di codice alla fine del rendering del template, oppure, se inserita all'interno del corpo di una funzione, di eseguire il codice quando la funzione sta per ritornare.

<pre class='example'>
{% file := openFile("test.txt) %}
{% <b>defer</b> file.Close() %}
{{ file.Content() }}
<hr>Contenuto del file......fine.
Il file test.txt è stato chiuso correttamente
</pre>


Se l'istruzione **defer** viene eseguita più volte, vengono eseguite per prime le chiamate di funzioni _deferred_ per ultime. Per maggiori informazioni si rimanda alla documentazione di Go relativa allo [statement defer](https://golang.org/doc/effective_go.html#defer).

#### Builtin recover

Il template in Scriggo mette a disposizione la builtin `recover`, il cui comportamento è identico all'ononima builtin presente in Go.
Si rimanda quindi alla documentazione di quest'ultima:

- [Blog di Go - Defer, Panic and Recover](https://blog.golang.org/defer-panic-and-recover)
- [Effective Go - Recover](https://golang.org/doc/effective_go.html#recover)
- [Specifiche di Go - Gestione dei panic](https://golang.org/ref/spec#Handling_panics)