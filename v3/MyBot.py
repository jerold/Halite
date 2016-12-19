from hlt import *
from networking import *

myID, gameMap = getInit()
sendInit("BrevityBot")

class Cell:
    def __init__(self, direction, site, location):
        self.direction = direction
        self.site = site
        self.location = location

def heuristic(cell):
    if cell.site.owner == 0 and cell.site.strength:
        return cell.site.production / cell.site.strength
    else:
        totalDamage = 0
        for d in CARDINALS:
            site = gameMap.getSite(cell.location, d)
            if site.owner != 0 and site.owner != myID:
                totalDamage += site.strength
        return totalDamage

def findNearestEnemyDirection(location):
    direction = SOUTH
    # don't get stuck in an infinite loop
    maxDistance = min(gameMap.width, gameMap.height) / 2
    for d in CARDINALS:
        distance = 0
        current = location
        site = gameMap.getSite(current, d)
        while site.owner == myID and distance < maxDistance:
            distance = distance + 1
            current = gameMap.getLocation(current, d)
            site = gameMap.getSite(current)
        if distance < maxDistance:
            direction = d
            maxDistance = distance
    return direction


def move(location):
    site = gameMap.getSite(location)
    border = False

    # don't attack squares we can't take. Pick strongest target
    target = {}
    for d in CARDINALS:
        cell = Cell(d, gameMap.getSite(location, d), gameMap.getLocation(location, d))
        if cell.site.owner != myID:
            border = True
            if not target or heuristic(cell) > heuristic(target):
                target = cell
    if target and target.site.strength < site.strength:
        return Move(location, target.direction);

    # don't move more than we have to
    if site.strength < site.production * 5:
        return Move(location, STILL)

    # if not on the border
    if not border:
        return Move(location, findNearestEnemyDirection(location));

    # wait until we can attack
    return Move(location, STILL)

while True:
    moves = []
    gameMap = getFrame()
    for y in range(gameMap.height):
        for x in range(gameMap.width):
            location = Location(x, y)
            if gameMap.getSite(location).owner == myID:
                moves.append(move(location))
    sendFrame(moves)
