Nginx, Spring Boot, Express ou Django : tous reposent sur la même base. Un programme ouvre un port, attend qu'un client se connecte, lit ce qu'il envoie et lui répond. Les frameworks rendent cette mécanique invisible, et c'est très bien en production. Mais tant qu'on ne l'a jamais vue, un « address already in use », un « connection reset by peer » ou un serveur qui refuse de s'arrêter restent des mystères.

Cette série construit un serveur web à partir de zéro, en Go, pour rendre chaque étape visible. Ce premier article couvre la toute première fonctionnalité : **accepter des connexions TCP**. On y suit le raisonnement complet, des questions qu'on se pose avant d'écrire une ligne jusqu'aux tests, en passant par les choix d'implémentation et leurs raisons.

## Pourquoi « Kross » ?

Pendant dix saisons au Real Madrid, Toni Kroos a tenu le milieu de terrain avec une spécialité : recevoir le ballon, lever la tête, et le rendre au bon endroit, au bon moment. Précis, régulier, rarement pris en défaut.

Un serveur web fait le même travail. Il reçoit une requête, comprend ce qu'on lui demande et renvoie la bonne réponse, sans jamais faire tomber le jeu. Le projet s'appelle donc **Kross**, un clin d'œil au numéro 8 du Real.

## Le point de départ : ce que fait vraiment un serveur web

Un serveur web est un programme qui attend. Il s'installe sur une adresse et un port, par exemple `127.0.0.1:8080`, et patiente jusqu'à ce qu'un client se présente. Quand un client arrive, les deux ouvrent une **connexion TCP** : un canal fiable dans lequel les octets arrivent dans l'ordre et sans perte. Le client y écrit sa requête, le serveur la lit, la traite et écrit sa réponse dans le même canal.

Ce qu'on appelle HTTP n'est que le contenu de ce canal, et c'est du texte. Voici, mot pour mot, ce que `curl` envoie quand on tape `curl http://127.0.0.1:8080/` :

```http
GET / HTTP/1.1
Host: 127.0.0.1:8080
User-Agent: curl/8.5.0
Accept: */*
```

Une ligne de requête, quelques en-têtes, une ligne vide. Rien de magique. Mais pour lire ces quatre lignes, il faut d'abord qu'une connexion existe. Avant de parler HTTP, le serveur doit donc maîtriser TCP : ouvrir un port, accepter un client, refermer la connexion. C'est l'objet de cette première partie.

> [!WARNING] Règle du jeu pour toute la série : on n'utilise pas le paquet `net/http`. On s'appuie uniquement sur les appels système et sur ce que le runtime de Go offre de plus bas niveau.

## La démarche : définir la fonctionnalité avant de coder

La première étape n'est pas d'écrire du code, mais de fixer ce que « terminé » veut dire. Pour cette partie, la définition tient en quatre points. Kross doit :

- écouter sur une adresse et un port configurables ;
- accepter les connexions entrantes, une par une, puis les refermer ;
- s'arrêter proprement quand on appuie sur Ctrl+C ;
- être couvert par des tests automatisés.

Ce périmètre est volontairement étroit. Il ne lit pas encore la requête, il ne répond rien et il ne sert qu'un client à la fois. Chaque limite deviendra le sujet d'une partie suivante. L'avantage d'un périmètre aussi net, c'est qu'on sait exactement quand s'arrêter.

### Les questions à se poser

Une fois l'objectif posé, quelques questions orientent tout le reste du découpage :

1. **Qu'est-ce qui doit être réglable ?** L'adresse, le port, et la taille de la file d'attente des connexions (on y revient plus loin). Ces réglages doivent être validés avant de toucher au réseau.
2. **Qui possède le socket ?** Un seul composant doit parler au noyau, pour que les détails système ne se diffusent pas partout dans le code.
3. **Que représente un client accepté ?** Un objet à nous, qui pourra s'enrichir plus tard sans que le reste du serveur change.
4. **Comment tester sans réseau ?** Le serveur ne doit pas dépendre d'une implémentation concrète, mais d'un contrat qu'on peut simuler.
5. **Comment s'arrêter proprement ?** Un serveur passe sa vie bloqué à attendre un client. Il faut un moyen fiable de le débloquer.

### Le découpage qui en découle

Chaque question donne un composant, avec une seule responsabilité :

- **ServerConfig** : l'adresse, le port et la taille de la file d'attente, avec leur validation.
- **ClientConnection** : une connexion acceptée, qui sait lire, écrire et se fermer.
- **Listener** : le contrat d'écoute (`Bind`, `Listen`, `Accept`, `Close`), et **TcpListener**, son implémentation avec les appels système.
- **Server** : l'orchestrateur, qui enchaîne la mise en écoute, la boucle d'acceptation et l'arrêt.

Le diagramme suivant montre comment ces composants collaborent pendant toute la vie du serveur :

![Diagramme de séquence de Kross : configuration, bind, listen, boucle d'acceptation, fermeture](https://raw.githubusercontent.com/brandonkamga237/kross/master/docs/articles/partie-1/images/lifecycle.png "Le cycle de vie de Kross dans cette première partie.")

Il se lit en trois temps :

1. **La préparation.** Le serveur récupère l'hôte, le port et le backlog, crée le `TcpListener`, puis appelle `bind()` et `listen()`. À la fin de cette phase, le port est ouvert et les clients peuvent se connecter.
2. **La boucle.** Tant que le serveur tourne, il appelle `accept()`. Chaque connexion d'un client est enveloppée dans une `ClientConnection`, puis rendue au serveur qui la traite.
3. **La fermeture.** À l'arrêt, `close()` libère le port et fait sortir le serveur de sa boucle.

L'implémentation suit l'ordre des dépendances : d'abord les briques qui ne dépendent de rien (la configuration, la connexion), puis le contrat d'écoute et son implémentation, enfin le serveur et le point d'entrée. Chaque étape ci-dessous correspond à un commit de la branche, accompagné de ses tests.

## Étape 1 : la configuration

Trois réglages suffisent pour démarrer, avec des valeurs par défaut raisonnables :

```go
type ServerConfig struct {
	Host    string
	Port    int
	Backlog int
}

func Default() ServerConfig {
	return ServerConfig{
		Host:    "127.0.0.1",
		Port:    8080,
		Backlog: 128,
	}
}

func (c ServerConfig) Validate() error {
	if c.Port < 0 || c.Port > 65535 {
		return fmt.Errorf("port out of range: %d", c.Port)
	}
	if c.Backlog < 1 {
		return fmt.Errorf("backlog must be at least 1: %d", c.Backlog)
	}
	return nil
}
```

Deux choix méritent une explication. Le port est codé sur 16 bits dans l'en-tête TCP, d'où la borne à 65535. Le port 0, lui, est volontairement accepté : il demande au noyau de choisir un port libre. C'est inutile en production, mais précieux dans les tests, où plusieurs serveurs doivent pouvoir démarrer sans se gêner.

L'hôte par défaut est `127.0.0.1`, l'adresse de boucle locale : le serveur n'est joignable que depuis la machine elle-même. Écouter sur `0.0.0.0` l'ouvrirait à tout le réseau, ce qui doit rester un choix explicite.

## Étape 2 : la connexion cliente

Avant même d'ouvrir un port, on définit ce que représente un client accepté. Go fournit déjà `net.Conn`, mais on l'enveloppe dans un type à nous :

```go
type ClientConnection struct {
	raw net.Conn
}

func (c *ClientConnection) Read(p []byte) (int, error) {
	return c.raw.Read(p)
}

func (c *ClientConnection) Write(p []byte) (int, error) {
	return c.raw.Write(p)
}
```

Aujourd'hui, ce type ne fait que transmettre les appels. L'intérêt est ailleurs : c'est lui qui accueillera plus tard les délais de lecture, la mise en mémoire tampon ou les statistiques. Comme il implémente `io.ReadWriteCloser`, il fonctionnera aussi avec tous les outils de la bibliothèque standard qui lisent ou écrivent des flux.

## Étape 3 : le contrat d'écoute

Le serveur ne doit pas savoir comment on écoute, seulement qu'on peut le faire. On formalise donc le contrat dans une interface :

```go
type Listener interface {
	Bind() error
	Listen() error
	Accept() (*conn.ClientConnection, error)
	Close() error
}
```

Les quatre méthodes reprennent les étapes du cycle de vie. Ce contrat est la réponse à la question « comment tester sans réseau ? » : dans les tests, on donnera au serveur un faux listener qui respecte la même interface.

## Étape 4 : Bind, créer le socket et lui donner une adresse

On entre maintenant dans le cœur du sujet. Pour accepter des connexions, un programme fait toujours quatre appels au noyau, dans cet ordre :

1. `socket` : crée une extrémité de connexion TCP. Le noyau répond par un **descripteur de fichier**, un simple entier qui désigne ce socket.
2. `bind` : associe ce socket à une adresse et un port.
3. `listen` : le passe en mode écoute et crée la file d'attente des connexions.
4. `accept` : retire de cette file la prochaine connexion établie.

En Go, `net.Listen("tcp", ":8080")` enchaîne les trois premiers en une ligne. C'est parfait en production, mais ça cache tout. Surtout, Go choisit lui-même la taille de la file d'attente à partir d'un réglage du système (sur Linux, `/proc/sys/net/core/somaxconn`). Kross veut décider de cette taille et voir chaque étape passer. Il fait donc les appels lui-même, avec le paquet `syscall`. Voici `Bind`, qui couvre les deux premiers :

```go
func (l *TcpListener) Bind() error {
	ip := net.ParseIP(l.host).To4()
	if ip == nil {
		return fmt.Errorf("invalid IPv4 address: %q", l.host)
	}

	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		return fmt.Errorf("socket: %w", err)
	}

	// Allows restarting without waiting for TIME_WAIT to expire.
	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		syscall.Close(fd)
		return fmt.Errorf("setsockopt: %w", err)
	}

	sa := &syscall.SockaddrInet4{Port: l.port}
	copy(sa.Addr[:], ip)

	if err := syscall.Bind(fd, sa); err != nil {
		syscall.Close(fd)
		return fmt.Errorf("bind %s:%d: %w", l.host, l.port, err)
	}

	l.fd = fd
	return nil
}
```

Trois points à retenir dans ce code :

- `AF_INET` désigne IPv4 et `SOCK_STREAM` désigne TCP. Kross ne gère qu'IPv4 pour l'instant, d'où la conversion `To4()` qui rejette tout le reste dès le départ.
- À chaque erreur, le descripteur est refermé avant de sortir. Sans cette précaution, chaque démarrage raté laisserait un socket ouvert jusqu'à la fin du processus.
- L'option `SO_REUSEADDR` est posée avant `bind`, car c'est au moment de `bind` que le noyau vérifie si l'adresse est disponible.

> [!TIP] Sans `SO_REUSEADDR`, arrêter Kross puis le relancer aussitôt échoue souvent avec « bind: address already in use ». Après une fermeture, le noyau garde l'adresse réservée un moment, dans l'état TIME_WAIT, pour absorber les derniers paquets en retard. Cette option l'autorise à réutiliser l'adresse immédiatement.

## Étape 5 : Listen, ouvrir la file d'attente

`listen` transforme le socket en socket d'écoute. Son paramètre principal est le **backlog** : la taille de la file où le noyau range les connexions déjà établies, en attendant que le programme vienne les chercher avec `accept`.

Un point surprend souvent : la poignée de main TCP en trois temps (SYN, SYN-ACK, ACK) n'implique pas le programme. Le noyau la termine seul, puis range la connexion prête dans la file. C'est pour cela qu'un client peut se connecter à un serveur occupé : sa connexion attend dans la file. Si la file est pleine, les nouveaux clients patientent ou échouent. Avec 128 places, Kross a de la marge.

```go
func (l *TcpListener) Listen() error {
	if l.fd < 0 {
		return errors.New("listen called before bind")
	}

	if err := syscall.Listen(l.fd, l.backlog); err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	// os.NewFile takes ownership of the descriptor.
	f := os.NewFile(uintptr(l.fd), "tcp-listener")
	l.fd = -1

	ln, err := net.FileListener(f)
	f.Close() // net.FileListener duplicates the descriptor, so f can be closed
	if err != nil {
		return fmt.Errorf("convert to net.Listener: %w", err)
	}

	l.ln = ln
	return nil
}
```

La garde du début protège l'ordre des appels : écouter avant d'avoir une adresse n'a pas de sens, et l'erreur est explicite plutôt qu'un code d'erreur système obscur.

La seconde moitié de la fonction est la décision la plus subtile de cette partie. On pourrait continuer avec `syscall.Accept`, mais cet appel bloquerait un thread du système à chaque attente. On confie donc le socket au runtime de Go avec `os.NewFile`, puis `net.FileListener`. À partir de là, l'attente d'un client passe par le mécanisme de Go (epoll sur Linux, kqueue sur macOS) : une goroutine qui attend ne coûte presque rien. On a fait nous-mêmes tout ce qui comptait pour comprendre, puis on s'appuie sur la mécanique du runtime, qui fait très bien ce travail.

Attention à la propriété du descripteur. `os.NewFile` en devient propriétaire, donc `l.fd` passe à -1 pour qu'on ne le referme jamais deux fois. `net.FileListener` en crée une copie, et on peut fermer l'original sans danger.

> [!TIP] Tu peux voir la file d'attente de tes propres yeux. Lance Kross, puis tape `ss -ltn` (sur macOS : `netstat -an | grep LISTEN`). Pour un socket en écoute, la colonne Send-Q affiche la taille du backlog et Recv-Q le nombre de connexions en attente.

## Étape 6 : Accept, recevoir un client

```go
func (l *TcpListener) Accept() (*conn.ClientConnection, error) {
	if l.ln == nil {
		return nil, errors.New("accept called before listen")
	}

	raw, err := l.ln.Accept()
	if err != nil {
		return nil, err
	}
	return conn.New(raw), nil
}
```

`Accept` bloque tant qu'aucune connexion n'attend dans la file. Dès qu'il y en a une, il la retire et l'enveloppe dans une `ClientConnection`. C'est l'étape « create ClientConnection(socket) » du diagramme.

Deux méthodes complètent le listener. `Close` ferme le socket d'écoute : c'est elle qui permettra l'arrêt propre. `Addr` renvoie l'adresse réellement utilisée, indispensable quand on a demandé le port 0 et qu'on veut savoir lequel le noyau a choisi.

## Étape 7 : le serveur, orchestrer le cycle de vie

Toutes les briques sont prêtes. Le serveur n'a plus qu'à les enchaîner :

```go
func (s *Server) Start() error {
	if err := s.listener.Bind(); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	if err := s.listener.Listen(); err != nil {
		s.listener.Close()
		return fmt.Errorf("start: %w", err)
	}
	return s.acceptLoop()
}

func (s *Server) Stop() error {
	return s.listener.Close()
}

func (s *Server) acceptLoop() error {
	for {
		c, err := s.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("accept: %w", err)
		}
		s.handle(c)
	}
}

func (s *Server) handle(c *conn.ClientConnection) {
	defer c.Close()
}
```

`Start` suit exactement le diagramme : `Bind`, `Listen`, puis la boucle. Si `Listen` échoue, le socket déjà créé par `Bind` est refermé avant de sortir. Pour l'instant, `handle` se contente de fermer la connexion : c'est là que viendra la lecture des requêtes.

Reste la dernière question : comment sortir d'une boucle bloquée dans `Accept` ? La réponse tient dans le comportement du socket. Fermer le socket d'écoute débloque l'appel `Accept` en cours, qui renvoie l'erreur `net.ErrClosed`. La boucle reconnaît cette erreur précise comme un arrêt normal et renvoie `nil`. Toute autre erreur reste une vraie erreur. `Stop` n'a donc qu'une ligne : fermer le listener.

## Étape 8 : le point d'entrée et l'arrêt propre

Le programme principal assemble le tout. Il lit les options de la ligne de commande, valide la configuration, crée le listener et le serveur, puis attend le signal d'arrêt :

```go
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()

l := listener.NewTcpListener(cfg.Host, cfg.Port, cfg.Backlog)
srv := server.New(cfg, l)

go func() {
	<-ctx.Done()
	if err := srv.Stop(); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}()

log.Printf("listening on %s:%d (backlog %d)", cfg.Host, cfg.Port, cfg.Backlog)
if err := srv.Start(); err != nil {
	log.Fatalf("server: %v", err)
}
log.Println("server stopped")
```

`Start` bloque le fil principal tant que le serveur tourne. L'arrêt passe donc par une goroutine séparée qui attend le signal. `SIGINT` correspond à Ctrl+C et `SIGTERM` au signal qu'envoient les gestionnaires de services comme systemd ou Docker. À la réception de l'un des deux, la goroutine appelle `Stop`, `Accept` renvoie `net.ErrClosed`, `Start` rend la main et le programme se termine normalement.

## Les tests

Un serveur réseau se vérifie mal à la main : il faut lancer le programme, ouvrir un second terminal, recommencer à chaque modification. Kross a donc ses tests dès la première fonctionnalité, lancés avec `make test`, qui active le détecteur de concurrence de Go (`-race`).

Le test du serveur illustre l'intérêt de l'interface `Listener`. On lui donne un faux listener dont `Accept` se comporte exactement comme le vrai : il bloque jusqu'à la fermeture, puis renvoie `net.ErrClosed`.

```go
func (f *fakeListener) Accept() (*conn.ClientConnection, error) {
	<-f.closed
	return nil, net.ErrClosed
}
```

Le test démarre le serveur dans une goroutine, attend que `Listen` ait été appelé, puis appelle `Stop`. Il vérifie que `Start` rend la main sans erreur en moins d'une seconde, et que `Bind` et `Listen` ont bien été appelés avant `Accept`. Le serveur ne voit pas la différence avec un vrai listener, et le test n'ouvre aucun port.

Le `TcpListener`, lui, est testé avec un vrai réseau. Le test le démarre sur le port 0, récupère le port choisi par le noyau grâce à `Addr`, s'y connecte avec `net.Dial` et vérifie qu'`Accept` renvoie bien un client. Deux autres tests vérifient les erreurs : écouter avant `bind` doit échouer, tout comme s'attacher à une adresse invalide. Enfin, la `ClientConnection` est testée avec `net.Pipe`, une connexion simulée en mémoire : ce qui est écrit d'un côté doit arriver intact de l'autre.

## On l'essaie

```bash
git clone -b feature/listening-socket https://github.com/brandonkamga237/kross.git
cd kross
make build
./bin/webserver -port 8099
```

Kross s'annonce :

```bash
2026/10/10 01:46:53 listening on 127.0.0.1:8099 (backlog 128)
```

Dans un second terminal, `ss` montre le socket en écoute. La colonne Send-Q affiche les 128 places de la file, et Recv-Q confirme qu'aucune connexion n'attend :

```bash
$ ss -ltn 'sport = :8099'
State  Recv-Q Send-Q Local Address:Port Peer Address:Port
LISTEN 0      128        127.0.0.1:8099      0.0.0.0:*
```

On se connecte avec `nc` :

```bash
$ nc -v 127.0.0.1 8099
Connection to 127.0.0.1 8099 port [tcp/*] succeeded!
```

La connexion est acceptée, puis refermée par Kross. Essayons maintenant avec `curl`, qui envoie une vraie requête HTTP :

```bash
$ curl -v http://127.0.0.1:8099/
> GET / HTTP/1.1
> Host: 127.0.0.1:8099
> User-Agent: curl/8.5.0
> Accept: */*
>
* Recv failure: Connection reset by peer
curl: (56) Recv failure: Connection reset by peer
```

Ce message s'explique entièrement par ce qu'on a construit. `curl` a envoyé sa requête, mais Kross a fermé la connexion sans la lire. Quand un programme ferme un socket qui contient encore des données non lues, le noyau ne fait pas une fermeture normale (FIN) : il coupe la connexion avec un paquet RST, que le client reçoit comme un « reset ».

> [!NOTE] Ce n'est pas un bug, c'est le point de départ de la partie 2. Le message disparaîtra dès que Kross lira les requêtes avant de fermer la connexion.

De retour dans le premier terminal, un Ctrl+C, et Kross s'arrête proprement :

```bash
2026/10/10 01:46:54 server stopped
```

## Ce qu'il faut retenir

- Un serveur web commence par quatre appels système : `socket`, `bind`, `listen`, puis `accept` en boucle.
- La poignée de main TCP est l'affaire du noyau. Le programme récupère des connexions déjà établies, rangées dans une file d'attente.
- Le backlog fixe la taille de cette file. Kross le choisit lui-même, et `ss -ltn` permet de le voir.
- `SO_REUSEADDR` évite le « address already in use » au redémarrage.
- Fermer le socket d'écoute est le moyen le plus simple de sortir proprement d'une boucle bloquée dans `accept`.
- Fermer une connexion sans lire ce qu'elle contient provoque un « connection reset by peer » chez le client.
- Programmer contre une interface rend le serveur testable sans réseau.

## Les limites de cette version

Kross est volontairement minimal à ce stade :

- il traite les connexions une par une, ce qui ne pose pas de problème tant qu'il les referme aussitôt ;
- il ne gère qu'IPv4 ;
- il fonctionne sur Linux et macOS (sous Windows, il faut passer par WSL) ;
- il ne comprend pas encore un mot de HTTP.

## Dans la partie 2

Kross ouvre la porte, mais il ne laisse encore personne parler. La prochaine partie lit enfin les octets envoyés par le client : la ligne de requête, les en-têtes et la ligne vide qui marque leur fin. On les découpera proprement, et le « connection reset by peer » laissera place à une vraie conversation.

## Pour aller plus loin

[Le code de cette partie : branche feature/listening-socket](https://github.com/brandonkamga237/kross/tree/feature/listening-socket)

[listen(2) : le backlog et la file d'attente, page de manuel Linux](https://man7.org/linux/man-pages/man2/listen.2.html)

[accept(2) : la page de manuel Linux](https://man7.org/linux/man-pages/man2/accept.2.html)

[Beej's Guide to Network Programming](https://beej.us/guide/bgnet/)

[RFC 9293 : la spécification de TCP](https://www.rfc-editor.org/rfc/rfc9293)
