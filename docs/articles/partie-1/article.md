Nginx, Spring Boot ou Express reposent tous sur la même base : un programme qui écoute sur un port TCP et accepte des connexions. Cette série construit cette base à la main, en Go, sans le paquet `net/http`, pour comprendre ce que les frameworks font à notre place.

Le projet s'appelle **Kross**, en hommage à Toni Kroos, le milieu de terrain du Real Madrid qui recevait chaque ballon et le rendait au bon endroit. C'est exactement le rôle d'un serveur web : recevoir une requête et renvoyer la bonne réponse.

## Objectif de cette partie

Un serveur web échange du HTTP, mais HTTP circule à l'intérieur d'une connexion TCP. Avant de lire la moindre requête, le serveur doit donc savoir ouvrir un port et accepter des connexions. C'est le seul objectif de cette première partie.

À la fin, Kross :

- écoute sur une adresse et un port configurables ;
- accepte les connexions entrantes, une par une, puis les referme ;
- s'arrête proprement sur Ctrl+C ;
- est couvert par des tests.

La lecture des requêtes HTTP arrive dans la partie 2.

## L'architecture

Kross est découpé en quatre composants, chacun avec une seule responsabilité :

- **ServerConfig** : l'adresse, le port et la taille de la file d'attente des connexions (le *backlog*), avec leur validation.
- **TcpListener** : le socket d'écoute, construit avec les appels système du noyau.
- **ClientConnection** : une connexion acceptée, qui sait lire, écrire et se fermer.
- **Server** : orchestre le cycle de vie, de la mise en écoute jusqu'à l'arrêt.

Le diagramme suivant montre comment ils collaborent :

![Diagramme de séquence de Kross : configuration, bind, listen, boucle d'acceptation, fermeture](https://raw.githubusercontent.com/brandonkamga237/kross/master/docs/articles/partie-1/images/lifecycle.png "Le cycle de vie de Kross dans cette première partie.")

On peut le lire en trois temps :

1. **La préparation.** Le serveur récupère l'hôte, le port et le backlog, crée le `TcpListener`, puis appelle `bind()` et `listen()`. À partir de là, le port est ouvert.
2. **La boucle.** Tant que le serveur tourne, il appelle `accept()`. Chaque connexion d'un client est enveloppée dans une `ClientConnection` et rendue au serveur.
3. **La fermeture.** À l'arrêt, `close()` libère le port.

`Server` ne manipule pas `TcpListener` directement, mais une interface `Listener` qui expose ces quatre méthodes. C'est ce qui permet de tester le serveur sans réseau.

## Sous le capot : quatre appels système

Ouvrir un port d'écoute se fait en quatre appels au noyau :

1. `socket` crée une extrémité de connexion TCP et renvoie un descripteur de fichier.
2. `bind` associe ce socket à une adresse et un port.
3. `listen` le passe en mode écoute et fixe la taille du backlog.
4. `accept` retire la prochaine connexion établie de la file d'attente.

Go propose `net.Listen`, qui enchaîne les trois premiers appels en une ligne. Mais il choisit lui-même la taille du backlog à partir d'un réglage du système. Kross fait donc les appels à la main, avec le paquet `syscall`. Voici l'essentiel, sans la gestion des erreurs :

```go
fd, _ := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)

// Allows restarting without waiting for TIME_WAIT to expire.
syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)

sa := &syscall.SockaddrInet4{Port: l.port}
copy(sa.Addr[:], ip)
syscall.Bind(fd, sa)

syscall.Listen(fd, l.backlog)
```

Dans le code complet, chaque échec referme le descripteur avant de renvoyer l'erreur.

> [!TIP] Sans l'option `SO_REUSEADDR`, relancer le serveur juste après l'avoir arrêté échoue souvent avec « address already in use » : le noyau garde l'adresse réservée quelques instants après une fermeture (état TIME_WAIT).

Une fois le socket en écoute, Kross le confie au runtime de Go avec `os.NewFile` et `net.FileListener`. Les appels suivants à `Accept` profitent alors du mécanisme d'attente de Go (epoll sur Linux, kqueue sur macOS) : une goroutine qui attend un client ne bloque pas un thread du système.

La poignée de main TCP entre le client et le serveur est entièrement gérée par le noyau. Quand `accept` est appelé, la connexion est déjà établie et attend dans la file.

## La boucle d'acceptation

Le cœur du serveur tient en quelques lignes :

```go
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
```

Pour l'instant, `handle` se contente de fermer la connexion.

L'arrêt repose sur un comportement simple : fermer le socket d'écoute débloque l'appel `Accept` en cours, qui renvoie `net.ErrClosed`. Le programme principal attend le signal Ctrl+C, appelle `Stop`, qui ferme le listener, et la boucle se termine sans erreur.

## Essayer Kross

```bash
git clone https://github.com/brandonkamga237/kross.git
cd kross
make build
./bin/webserver -port 8099
```

Dans un second terminal, `nc` confirme que la connexion est acceptée :

```bash
$ nc -v 127.0.0.1 8099
Connection to 127.0.0.1 8099 port [tcp/*] succeeded!
```

Avec `curl`, le résultat est différent :

```bash
$ curl http://127.0.0.1:8099/
curl: (56) Recv failure: Connection reset by peer
```

`curl` envoie une requête HTTP, que Kross ne lit pas encore. Quand un programme ferme un socket qui contient encore des données non lues, le noyau interrompt la connexion avec un paquet RST au lieu d'une fermeture normale. C'est attendu à ce stade : ce comportement disparaîtra dès que Kross lira les requêtes.

## Les tests

Chaque composant est testé, avec `make test` et le détecteur de concurrence de Go :

- le `TcpListener` démarre sur le port 0, que le noyau remplace par un port libre, puis accepte une vraie connexion ;
- le `Server` reçoit un faux `Listener` qui vérifie l'ordre des appels et l'arrêt propre ;
- la `ClientConnection` est testée avec `net.Pipe`, une connexion simulée en mémoire.

## À retenir

- Un serveur web commence par quatre appels système : `socket`, `bind`, `listen`, puis `accept` en boucle.
- La poignée de main TCP est gérée par le noyau ; le programme récupère des connexions déjà établies.
- Le backlog est la file d'attente de ces connexions, et Kross en contrôle la taille.
- Fermer le socket d'écoute est la façon la plus simple d'arrêter proprement la boucle d'acceptation.

## Dans la partie 2

Kross accepte des connexions mais ne lit encore rien. La prochaine partie lit et découpe une requête HTTP : la ligne de requête, les en-têtes et la fin de l'en-tête.

## Pour aller plus loin

[Le code de Kross sur GitHub](https://github.com/brandonkamga237/kross)

[listen(2) : le backlog et la file d'attente](https://man7.org/linux/man-pages/man2/listen.2.html)

[Beej's Guide to Network Programming](https://beej.us/guide/bgnet/)
